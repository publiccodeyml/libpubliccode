package netutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIsPublicIP(t *testing.T) {
	cases := map[string]bool{
		"8.8.8.8":         true,
		"1.1.1.1":         true,
		"93.184.216.34":   true,
		"127.0.0.1":       false, // loopback
		"10.0.0.1":        false, // private
		"192.168.1.1":     false, // private
		"172.16.0.1":      false, // private
		"169.254.169.254": false, // link-local (cloud metadata)
		"100.64.0.1":      false, // CGNAT
		"0.0.0.0":         false, // unspecified
		"::1":             false, // IPv6 loopback
		"fc00::1":         false, // IPv6 ULA
		"fe80::1":         false, // IPv6 link-local
	}

	for addr, want := range cases {
		got := isPublicIP(net.ParseIP(addr))
		if got != want {
			t.Errorf("isPublicIP(%s) = %v, want %v", addr, got, want)
		}
	}
}

// TestSafeHTTPClientBlocksPrivate proves the default (allowPrivate=false) client
// refuses to connect to a loopback address, defeating SSRF, while the opt-out
// client can reach it.
func TestSafeHTTPClientBlocksPrivate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	blocked := SafeHTTPClient(5*time.Second, false)
	if _, err := getBody(t.Context(), blocked, srv.URL); err == nil {
		t.Fatal("expected the SSRF guard to block the loopback request")
	} else if !errors.Is(err, ErrBlockedAddress) && !strings.Contains(err.Error(), "non-public") {
		t.Errorf("expected a blocked-address error, got: %v", err)
	}

	allowed := SafeHTTPClient(5*time.Second, true)
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
	resp, err := allowed.Do(req)
	if err != nil {
		t.Fatalf("expected the opt-out client to reach the server, got: %v", err)
	}
	_ = resp.Body.Close()
}

// roundTripFunc lets us stub a RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newResponse(body string, contentLength int64) *http.Response {
	return &http.Response{
		StatusCode:    http.StatusOK,
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: contentLength,
		Header:        make(http.Header),
	}
}

func TestResponseSizeLimitOnRead(t *testing.T) {
	// Unknown Content-Length, body larger than max: the limit must trigger while
	// reading the body.
	transport := &safeTransport{
		base: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return newResponse(strings.Repeat("A", 100), -1), nil
		}),
		max: 10,
	}

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com", nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}

	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Errorf("expected ErrResponseTooLarge while reading, got: %v", err)
	}
}

func TestResponseSizeLimitOnContentLength(t *testing.T) {
	// Advertised Content-Length over the limit must be rejected up front.
	transport := &safeTransport{
		base: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return newResponse("small", 1000), nil
		}),
		max: 10,
	}

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com", nil)
	resp, err := transport.RoundTrip(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Errorf("expected ErrResponseTooLarge from Content-Length check, got: %v", err)
	}
}

func TestResponseUnderLimitIsReadFully(t *testing.T) {
	const body = "within limit"
	transport := &safeTransport{
		base: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return newResponse(body, int64(len(body))), nil
		}),
		max: MaxResponseBytes,
	}

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com", nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer resp.Body.Close()
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if string(got) != body {
		t.Errorf("body mismatch: got %q, want %q", got, body)
	}
}

func TestUserAgentTransportSetsTheDefault(t *testing.T) {
	const userAgent = "libpubliccode/1.2.3 (+https://example.org)"

	var got string
	transport := &userAgentTransport{
		base: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Header.Get("User-Agent")

			return newResponse("", 0), nil
		}),
		userAgent: userAgent,
	}

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com", nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if got != userAgent {
		t.Errorf("User-Agent sent: got %q, want %q", got, userAgent)
	}

	// A RoundTripper must not modify the request it is handed.
	if _, set := req.Header["User-Agent"]; set {
		t.Errorf("the original request was modified: %q", req.Header.Get("User-Agent"))
	}
}

func TestUserAgentTransportKeepsTheCallerHeader(t *testing.T) {
	cases := map[string]string{
		"caller sets its own": "harvester/2.0",
		"caller clears it":    "",
	}

	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			var got string
			var seen bool

			transport := &userAgentTransport{
				base: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					got, seen = r.Header.Get("User-Agent"), true

					return newResponse("", 0), nil
				}),
				userAgent: "libpubliccode/1.2.3",
			}

			req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com", nil)
			req.Header["User-Agent"] = []string{want}

			resp, err := transport.RoundTrip(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			if !seen {
				t.Fatal("the base transport was not called")
			}
			if got != want {
				t.Errorf("User-Agent sent: got %q, want %q", got, want)
			}
		})
	}
}

// TestSafeHTTPClientSendsUserAgent checks the User-Agent all the way down to the
// wire, and that the library's own is sent when none is given.
func TestSafeHTTPClientSendsUserAgent(t *testing.T) {
	const userAgent = "libpubliccode/1.2.3 (+https://example.org)"

	// The server echoes back what it received, so there is nothing shared
	// between the handler and the test.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, r.UserAgent()) //nolint:gosec // This local test server echoes a header as plain text.
	}))
	defer srv.Close()

	got, err := getBody(t.Context(), SafeHTTPClient(5*time.Second, true, WithUserAgent(userAgent)), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != userAgent {
		t.Errorf("User-Agent received by the server: got %q, want %q", got, userAgent)
	}

	if got, err = getBody(t.Context(), SafeHTTPClient(5*time.Second, true), srv.URL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != UserAgent() {
		t.Errorf("User-Agent received by the server: got %q, want the default %q", got, UserAgent())
	}
}

// getBody GETs url and returns the response body as a string.
func getBody(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("get body: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	return string(body), err
}
