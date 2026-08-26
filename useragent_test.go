package publiccode

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"
	"testing"
)

// userAgentRecorder is a server recording the User-Agent of every request it
// receives.
type userAgentRecorder struct {
	*httptest.Server

	mu   sync.Mutex
	seen []string
}

// newUserAgentRecorder starts a recording server, stopped when the test ends.
func newUserAgentRecorder(t *testing.T) *userAgentRecorder {
	t.Helper()

	recorder := &userAgentRecorder{}
	recorder.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder.mu.Lock()
		recorder.seen = append(recorder.seen, r.UserAgent())
		recorder.mu.Unlock()

		w.Header().Set("Content-Type", "text/yaml")
		_, _ = w.Write([]byte("publiccodeYmlVersion: \"0.4\"\n"))
	}))

	t.Cleanup(recorder.Close)

	return recorder
}

// userAgents returns the User-Agents recorded so far.
func (r *userAgentRecorder) userAgents() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.seen)
}

// TestUserAgentIsSentOnEveryRequestPath covers the two ways the parser reaches
// the network: the request it builds itself in Parse(), and the ones built by
// the internal httpclient during the external checks.
func TestUserAgentIsSentOnEveryRequestPath(t *testing.T) {
	cases := map[string]string{
		"default":  UserAgent(),
		"override": "harvester/2.0 (+https://example.org)",
	}

	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			srv := newUserAgentRecorder(t)

			config := ParserConfig{AllowNetworkToPrivateHosts: true}
			if name == "override" {
				config.UserAgent = want
			}

			p, err := NewParser(config)
			if err != nil {
				t.Fatal(err)
			}

			// Parse() builds its request itself.
			_, _ = p.Parse(srv.URL + "/publiccode.yml")

			// The external checks go through the internal httpclient.
			u, err := url.Parse(srv.URL + "/anything")
			if err != nil {
				t.Fatal(err)
			}

			if _, err = p.isReachable(*u); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			seen := srv.userAgents()
			if len(seen) < 2 {
				t.Fatalf("expected both request paths to reach the server, got %d request(s)", len(seen))
			}

			for _, got := range seen {
				if got != want {
					t.Errorf("User-Agent received by the server: got %q, want %q", got, want)
				}
			}
		})
	}
}
