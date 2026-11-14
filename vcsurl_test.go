package publiccode

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestParserNetworkPolicyIsolation(t *testing.T) {
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	u, err := url.Parse(srv.URL + "/org/repo")
	if err != nil {
		t.Fatal(err)
	}
	strict, err := NewParser(ParserConfig{})
	if err != nil {
		t.Fatal(err)
	}
	permissive, err := NewParser(ParserConfig{AllowNetworkToPrivateHosts: true})
	if err != nil {
		t.Fatal(err)
	}
	base := &url.URL{Scheme: "file", Path: t.TempDir()}
	for _, validate := range []func() error{
		func() error { return validateFieldsV0(&PublicCodeV0{URL: (*URL)(u)}, strict, true, base) },
		func() error { return validateFieldsV1(&PublicCodeV1{URL: (*URL)(u)}, strict, true, base) },
	} {
		err := validate()
		if err == nil || !strings.Contains(err.Error(), "is not a valid code repository") {
			t.Fatalf("expected repository rejection, got %v", err)
		}
	}
	_, _ = strict.getRawRoot(u)
	_ = strict.toAbsoluteURL(srv.URL+"/org/repo/blob/main/logo.svg", base, true)
	if got := requests.Load(); got != 0 {
		t.Fatalf("strict parser sent %d private requests", got)
	}
	// Create a parser without AllowNetworkToPrivateHosts after the permissive
	// one. If NewParser changed the client shared by all parsers, the
	// permissive parser would no longer reach the local test server.
	if _, err := NewParser(ParserConfig{}); err != nil {
		t.Fatal(err)
	}
	if !permissive.isRepo(u) {
		t.Fatal("permissive parser rejected local repository")
	}
	if requests.Load() == 0 {
		t.Fatal("permissive parser did not reach server")
	}
}

func TestConcurrentParserNetworkPolicies(t *testing.T) {
	var unexpected atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.UserAgent() != "permissive-parser" {
			unexpected.Add(1)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	u, err := url.Parse(srv.URL + "/org/repo")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for _, allow := range []bool{false, true} {
		wg.Go(func() {
			for range 10 {
				agent := "strict-parser"
				if allow {
					agent = "permissive-parser"
				}
				p, err := NewParser(ParserConfig{AllowNetworkToPrivateHosts: allow, UserAgent: agent})
				if err != nil {
					t.Error(err)

					return
				}
				if got := p.isRepo(u); got != allow {
					t.Errorf("isRepo = %v, allow private = %v", got, allow)
				}
				_, _ = p.getRawRoot(u)
				_ = p.getRawFile(u)
			}
		})
	}
	wg.Wait()
	if got := unexpected.Load(); got != 0 {
		t.Fatalf("got %d requests with unexpected User-Agent", got)
	}
}
