package netutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

// MaxResponseBytes caps how many bytes the parser reads from a single remote
// resource. Without a cap an attacker-controlled URL in a publiccode.yml could
// point at an endless or huge response and exhaust the memory/disk of the host
// running the parser (e.g. the publiccode-crawler).
const MaxResponseBytes int64 = 20 << 20 // 20 MiB

// ErrBlockedAddress is returned when a request would connect to a non-public
// (loopback, private, link-local, ...) address. It protects against SSRF where
// a URL in an untrusted publiccode.yml is used to reach internal services or
// cloud metadata endpoints.
var ErrBlockedAddress = errors.New("blocked attempt to connect to a non-public address")

// ErrResponseTooLarge is returned when a remote response exceeds MaxResponseBytes.
var ErrResponseTooLarge = fmt.Errorf("response body exceeds the maximum allowed size of %d bytes", MaxResponseBytes)

// isPublicIP reports whether ip is a globally routable address the parser is
// allowed to reach. Loopback, unspecified, private (RFC 1918 / RFC 4193),
// link-local (incl. the 169.254.169.254 cloud metadata address), CGNAT and
// multicast ranges are refused.
func isPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() ||
		ip.IsInterfaceLocalMulticast() {
		return false
	}

	// 100.64.0.0/10 (CGNAT, RFC 6598) is not covered by IsPrivate().
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return false
		}
	}

	return true
}

// newDialControl returns a net.Dialer Control function that rejects connections
// to non-public addresses. It runs after DNS resolution, with the concrete IP
// about to be dialed, so it also defeats DNS-rebinding and is re-evaluated on
// every redirect hop.
func newDialControl() func(network, address string, c syscall.RawConn) error {
	return func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return fmt.Errorf("parsing dial address %q: %w", address, err)
		}

		ip := net.ParseIP(host)
		if !isPublicIP(ip) {
			return fmt.Errorf("%w: %s", ErrBlockedAddress, address)
		}

		return nil
	}
}

// proxyURLKey is the context key that passes the proxy URL chosen by
// proxyTransport to the transport that sends the request, so the proxy
// environment variables are read only once per request.
type proxyURLKey struct{}

// proxyTransport sends a request either directly or through the proxy set in
// HTTP_PROXY or HTTPS_PROXY.
//
// For a direct request the dialer sees the target IP and blocks non-public
// addresses. Through a proxy the dialer only sees the proxy IP, so
// proxyTransport resolves the target host itself and refuses the request when
// any of its addresses is non-public.
//
// The proxy resolves the host again and can get a different answer, for
// example from a different DNS server. Only filtering on the proxy itself
// blocks private targets in that case.
type proxyTransport struct {
	direct       http.RoundTripper
	proxied      http.RoundTripper
	proxy        func(*http.Request) (*url.URL, error)
	lookup       func(context.Context, string) ([]net.IPAddr, error)
	allowPrivate bool
}

func (t *proxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	proxy, err := t.proxy(req)
	if err != nil {
		return nil, err
	}

	if proxy == nil {
		return t.direct.RoundTrip(req)
	}

	if !t.allowPrivate {
		host := req.URL.Hostname()

		addresses, err := t.lookup(req.Context(), host)
		if err != nil {
			return nil, fmt.Errorf("resolving proxy target %q: %w", host, err)
		}

		if len(addresses) == 0 {
			return nil, fmt.Errorf("%w: %s resolved to no addresses", ErrBlockedAddress, host)
		}

		for _, address := range addresses {
			if !isPublicIP(address.IP) {
				return nil, fmt.Errorf("%w: %s (%s)", ErrBlockedAddress, host, address.IP)
			}
		}
	}

	req = req.Clone(context.WithValue(req.Context(), proxyURLKey{}, proxy))

	return t.proxied.RoundTrip(req)
}

// limitedBody wraps a response body and returns ErrResponseTooLarge once more
// than max bytes have been read, so a single response cannot exhaust memory.
type limitedBody struct {
	body io.ReadCloser
	max  int64
	read int64
}

func (l *limitedBody) Read(p []byte) (int, error) {
	if l.read > l.max {
		return 0, ErrResponseTooLarge
	}

	n, err := l.body.Read(p)
	l.read += int64(n)

	if l.read > l.max {
		return n, ErrResponseTooLarge
	}

	return n, err //nolint:wrapcheck // must forward io.EOF and other sentinels unwrapped
}

func (l *limitedBody) Close() error {
	return l.body.Close() //nolint:wrapcheck // transparent pass-through of the wrapped body
}

// safeTransport wraps a base RoundTripper enforcing the response size limit.
// Non-public addresses are blocked one layer below: by the dialer for direct
// requests and by proxyTransport for requests sent through a proxy.
type safeTransport struct {
	base http.RoundTripper
	max  int64
}

func (t *safeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Reject early when the server advertises an over-limit Content-Length.
	if resp.ContentLength > t.max {
		_ = resp.Body.Close()

		return nil, ErrResponseTooLarge
	}

	resp.Body = &limitedBody{body: resp.Body, max: t.max}

	return resp, nil
}

// userAgentTransport sets a User-Agent on the requests that don't carry one,
// since WAFs often block Go's default "Go-http-client/<version>".
type userAgentTransport struct {
	base      http.RoundTripper
	userAgent string
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if _, set := req.Header["User-Agent"]; !set && req.Header != nil {
		// A RoundTripper must not modify the request it is given, hence the clone.
		req = req.Clone(req.Context())
		req.Header.Set("User-Agent", t.userAgent)
	}

	return t.base.RoundTrip(req)
}

// Option configures the client built by [SafeHTTPClient].
type Option func(*options)

type options struct {
	userAgent string
}

// WithUserAgent sets the User-Agent sent on every request that doesn't already
// carry one, in place of [UserAgent]. An empty value keeps that default.
func WithUserAgent(userAgent string) Option {
	return func(o *options) {
		o.userAgent = userAgent
	}
}

// SafeHTTPClient builds an *http.Client hardened against SSRF and unbounded
// downloads. When allowPrivate is true the SSRF address filtering is disabled
// (used for trusted input and tests that target loopback servers); the response
// size limit is always enforced.
//
// A proxy set in HTTP_PROXY or HTTPS_PROXY is used even when its address is
// private. With a proxy set, the client checks the addresses of the target
// host before each request, redirects included. The proxy resolves the host
// again, so it needs its own filtering when its DNS answers can differ.
func SafeHTTPClient(timeout time.Duration, allowPrivate bool, opts ...Option) *http.Client {
	o := options{userAgent: UserAgent()}
	for _, opt := range opts {
		opt(&o)
	}

	if o.userAgent == "" {
		o.userAgent = UserAgent()
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	if !allowPrivate {
		dialer.Control = newDialControl()
	}

	transport := &http.Transport{
		// proxyTransport reads the proxy environment and picks the transport.
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// The proxy address comes from the environment of the process, not from the
	// publiccode.yml, so this dialer does not block private addresses.
	proxyDialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	proxied := transport.Clone()
	proxied.DialContext = proxyDialer.DialContext
	proxied.Proxy = func(req *http.Request) (*url.URL, error) {
		proxy, _ := req.Context().Value(proxyURLKey{}).(*url.URL)

		return proxy, nil
	}
	router := &proxyTransport{
		direct:       transport,
		proxied:      proxied,
		proxy:        http.ProxyFromEnvironment,
		lookup:       net.DefaultResolver.LookupIPAddr,
		allowPrivate: allowPrivate,
	}

	return &http.Client{
		Timeout: timeout,
		Transport: &userAgentTransport{
			base:      &safeTransport{base: router, max: MaxResponseBytes},
			userAgent: o.userAgent,
		},
	}
}
