package publiccode

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func newPrivateNetworkParser(t *testing.T) *Parser {
	t.Helper()

	p, err := NewParser(ParserConfig{DisableNetwork: true, AllowNetworkToPrivateHosts: true})
	if err != nil {
		t.Fatal(err)
	}

	return p
}

func validPubliccode(t *testing.T) []byte {
	t.Helper()

	content, err := os.ReadFile("testdata/v0/valid/valid.yml")
	if err != nil {
		t.Fatal(err)
	}

	return content
}

func TestParseRejectsFailedHTTPResponses(t *testing.T) {
	content := validPubliccode(t)

	for _, tc := range []struct {
		status int
		want   string
	}{
		{http.StatusNotFound, "404 Not Found"},
		{http.StatusInternalServerError, "500 Internal Server Error"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write(content)
			}))
			defer srv.Close()

			uri := srv.URL + "/publiccode.yml"

			_, err := newPrivateNetworkParser(t).Parse(uri)
			if err == nil {
				t.Fatal("expected an error for a failed download")
			}

			if !strings.Contains(err.Error(), uri) || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not name %s and %s", err, uri, tc.want)
			}
		})
	}
}

func TestParseHTTPSuccess(t *testing.T) {
	content := validPubliccode(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	if _, err := newPrivateNetworkParser(t).Parse(srv.URL + "/publiccode.yml"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseHTTPRedirect(t *testing.T) {
	content := validPubliccode(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/old.yml", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/publiccode.yml", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/gone.yml", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/missing.yml", http.StatusFound)
	})
	mux.HandleFunc("/publiccode.yml", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(content)
	})
	mux.HandleFunc("/missing.yml", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(content)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := newPrivateNetworkParser(t)

	if _, err := p.Parse(srv.URL + "/old.yml"); err != nil {
		t.Errorf("redirect to a valid file: unexpected error: %v", err)
	}

	_, err := p.Parse(srv.URL + "/gone.yml")
	if err == nil || !strings.Contains(err.Error(), "404 Not Found") {
		t.Errorf("redirect to a missing file: got %v, want a 404 error", err)
	}
}

type closeTracker struct {
	io.Reader
	closed bool
}

func (c *closeTracker) Close() error {
	c.closed = true

	return nil
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestParseHTTPClosesBody(t *testing.T) {
	content := validPubliccode(t)

	for _, status := range []int{http.StatusOK, http.StatusNotFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			body := &closeTracker{Reader: strings.NewReader(string(content))}

			p := newPrivateNetworkParser(t)
			p.client = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: status,
					Status:     http.StatusText(status),
					Body:       body,
					Request:    r,
				}, nil
			})}

			_, _ = p.Parse("https://example.org/publiccode.yml")

			if !body.closed {
				t.Error("response body was not closed")
			}
		})
	}
}
