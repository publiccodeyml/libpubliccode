package netutil

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// net/http reads the proxy environment variables once per process, so the test
// sets them in a subprocess.
func TestSafeHTTPClientProxyEnvironment(t *testing.T) {
	if os.Getenv("SAFEHTTP_PROXY_CHILD") == "1" {
		client := SafeHTTPClient(5*time.Second, false)
		body, err := getBody(client, "http://8.8.8.8/resource")
		if err != nil || body != "proxied" {
			t.Fatalf("public target through private proxy: body=%q, err=%v", body, err)
		}
		for _, target := range []string{"http://169.254.169.254/", "https://10.0.0.1/"} {
			if _, err := client.Get(target); !errors.Is(err, ErrBlockedAddress) {
				t.Fatalf("private target %s: %v", target, err)
			}
		}
		return
	}
	var requests atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.String() != "http://8.8.8.8/resource" {
			t.Errorf("unexpected proxy request: %s %s", r.Method, r.URL)
		}
		_, _ = io.WriteString(w, "proxied")
	}))
	defer proxy.Close()
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestSafeHTTPClientProxyEnvironment$")
	cmd.Env = append(os.Environ(), "SAFEHTTP_PROXY_CHILD=1", "HTTP_PROXY="+proxy.URL,
		"HTTPS_PROXY="+proxy.URL, "NO_PROXY=", "REQUEST_METHOD=")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("proxy environment subprocess: %v\n%s", err, output)
	}
	if requests.Load() != 1 {
		t.Fatalf("proxy received %d requests, want 1", requests.Load())
	}
}

func TestProxyTargetFiltering(t *testing.T) {
	for _, scheme := range []string{"http", "https"} {
		for _, tc := range []struct {
			name         string
			ips          []string
			lookupErr    error
			allowPrivate bool
			wantErr      bool
		}{
			{name: "public", ips: []string{"8.8.8.8", "2001:4860:4860::8888"}},
			{name: "private", ips: []string{"10.0.0.1"}, wantErr: true},
			{name: "mixed", ips: []string{"8.8.8.8", "::1"}, wantErr: true},
			{name: "empty", wantErr: true},
			{name: "DNS failure", lookupErr: errors.New("DNS failed"), wantErr: true},
			{name: "opt out", ips: []string{"10.0.0.1"}, allowPrivate: true},
		} {
			t.Run(scheme+"/"+tc.name, func(t *testing.T) {
				var calls int
				proxyURL, _ := url.Parse("http://127.0.0.1:1234")
				router := &proxyTransport{
					direct: roundTripFunc(func(*http.Request) (*http.Response, error) {
						t.Fatal("unexpected direct request")
						return nil, nil
					}),
					proxied: roundTripFunc(func(req *http.Request) (*http.Response, error) {
						calls++
						if req.Context().Value(proxyURLKey{}) != proxyURL {
							t.Error("selected proxy was not passed to transport")
						}
						return newResponse("ok", 2), nil
					}),
					proxy: http.ProxyURL(proxyURL),
					lookup: func(ctx context.Context, host string) ([]net.IPAddr, error) {
						if host != "target.test" {
							t.Errorf("resolved host %q", host)
						}
						if tc.allowPrivate {
							t.Error("opt out must skip DNS checks")
						}
						var addresses []net.IPAddr
						for _, ip := range tc.ips {
							addresses = append(addresses, net.IPAddr{IP: net.ParseIP(ip)})
						}
						return addresses, tc.lookupErr
					},
					allowPrivate: tc.allowPrivate,
				}
				req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, scheme+"://target.test/", nil)
				resp, err := router.RoundTrip(req)
				if resp != nil {
					_ = resp.Body.Close()
				}
				if (err != nil) != tc.wantErr {
					t.Fatalf("error=%v, wantErr=%v", err, tc.wantErr)
				}
				if tc.wantErr && tc.lookupErr == nil && !errors.Is(err, ErrBlockedAddress) {
					t.Fatalf("expected blocked address, got %v", err)
				}
				if tc.lookupErr != nil && !errors.Is(err, tc.lookupErr) {
					t.Fatalf("lost DNS error: %v", err)
				}
				if !tc.wantErr && calls != 1 {
					t.Fatalf("proxy calls=%d, want 1", calls)
				}
				if tc.wantErr && calls != 0 {
					t.Fatal("blocked target reached proxy")
				}
			})
		}
	}
}

func TestProxyRedirectAndBypass(t *testing.T) {
	var requests atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Redirect(w, r, "http://169.254.169.254/", http.StatusFound)
	}))
	defer proxy.Close()
	proxyURL, _ := url.Parse(proxy.URL)
	client := SafeHTTPClient(5*time.Second, false)
	router := client.Transport.(*userAgentTransport).base.(*safeTransport).base.(*proxyTransport)
	router.proxy = http.ProxyURL(proxyURL)
	if _, err := client.Get("http://8.8.8.8/"); !errors.Is(err, ErrBlockedAddress) {
		t.Fatalf("redirect to private target: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("proxy received %d requests", requests.Load())
	}
	// Without a proxy, as when NO_PROXY matches the host, the dialer must still
	// block private addresses.
	router.proxy = func(*http.Request) (*url.URL, error) { return nil, nil }
	if _, err := client.Get(proxy.URL); !errors.Is(err, ErrBlockedAddress) {
		t.Fatalf("direct private target: %v", err)
	}
}

func TestHTTPSProxyTarget(t *testing.T) {
	var connect atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect || r.Host != "8.8.8.8:443" {
			t.Errorf("unexpected CONNECT: %s %s", r.Method, r.Host)
		}
		connect.Add(1)
		// Answer CONNECT with an error, so the test needs no TLS server behind
		// the proxy.
		http.Error(w, "test proxy", http.StatusBadGateway)
	}))
	defer proxy.Close()
	proxyURL, _ := url.Parse(proxy.URL)
	client := SafeHTTPClient(5*time.Second, false)
	router := client.Transport.(*userAgentTransport).base.(*safeTransport).base.(*proxyTransport)
	router.proxy = http.ProxyURL(proxyURL)
	_, err := client.Get("https://8.8.8.8/")
	if err == nil || errors.Is(err, ErrBlockedAddress) || !strings.Contains(err.Error(), http.StatusText(http.StatusBadGateway)) {
		t.Fatalf("expected proxy response, got %v", err)
	}
	if connect.Load() != 1 {
		t.Fatalf("CONNECT count=%d", connect.Load())
	}
}
