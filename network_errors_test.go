package publiccode

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFailedNetworkChecksUnwrapToTheirCause(t *testing.T) {
	release := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rate-limited":
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
		case "/stalled":
			select {
			case <-r.Context().Done():
			case <-release:
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	content := validPubliccode(t)
	for old, path := range map[string]string{
		`url: "https://github.com/italia/developers.italia.it.git"`: `url: "` + srv.URL + `/repository"`,
		`landingURL: "https://developers.italia.it"`:                `landingURL: "` + srv.URL + `/rate-limited"`,
		`roadmap: "https://designers.italia.it/roadmap/"`:           `roadmap: "` + srv.URL + `/stalled"`,
	} {
		if !bytes.Contains(content, []byte(old)) {
			t.Fatalf("fixture has no %s", old)
		}

		content = bytes.Replace(content, []byte(old), []byte(path), 1)
	}

	content = bytes.Replace(content,
		[]byte(`documentation: "https://raw.githubusercontent.com/`),
		[]byte(`documentation: "`+srv.URL+`/missing/`), 1)

	p, err := NewParser(ParserConfig{
		BaseURL:                    "testdata/v0/valid/",
		Timeout:                    500 * time.Millisecond,
		AllowNetworkToPrivateHosts: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.ParseStream(bytes.NewReader(content))

	vr, ok := errors.AsType[ValidationResults](err)
	if !ok {
		t.Fatalf("got %v, want ValidationResults", err)
	}

	want := map[string]error{
		"landingURL":                   ErrRateLimited,
		"roadmap":                      ErrTimeout,
		"description.en.documentation": nil,
	}

	for key, cause := range want {
		found := false

		for _, result := range vr {
			valErr, ok := errors.AsType[ValidationError](result)
			if !ok || valErr.Key != key || !strings.Contains(valErr.Description, "not reachable") {
				continue
			}

			found = true

			if got := valErr.Unwrap(); !errors.Is(got, cause) {
				t.Errorf("%s: Unwrap() = %v, want %v", key, got, cause)
			}

			if cause != nil && !errors.Is(result, cause) {
				t.Errorf("%s: errors.Is(%v, %v) = false", key, result, cause)
			}

			if errors.Is(result, ErrRateLimited) && !errors.Is(cause, ErrRateLimited) {
				t.Errorf("%s: %v matches ErrRateLimited", key, result)
			}
		}

		if !found {
			t.Errorf("no unreachable error for %s in:\n%v", key, vr)
		}
	}
}

func TestLogoNetworkChecksUnwrapToTheirCause(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rate-limited.png" {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)

			return
		}
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	for _, tc := range []struct {
		name  string
		path  string
		cause error
	}{
		{"rate limited", "/rate-limited.png", ErrRateLimited},
		{"timeout", "/stalled.png", ErrTimeout},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewParser(ParserConfig{
				BaseURL:                    srv.URL + "/",
				Timeout:                    50 * time.Millisecond,
				AllowNetworkToPrivateHosts: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			content := "publiccodeYmlVersion: '0.8'\nlogo: " + srv.URL + tc.path +
				"\nmonochromeLogo: " + srv.URL + tc.path + "\n"
			_, err = p.ParseStream(strings.NewReader(content))
			vr, ok := errors.AsType[ValidationResults](err)
			if !ok {
				t.Fatalf("got %v, want ValidationResults", err)
			}
			for _, key := range []string{"logo", "monochromeLogo"} {
				found := false
				for _, result := range vr {
					valErr, ok := errors.AsType[ValidationError](result)
					if !ok || valErr.Key != key {
						continue
					}
					found = true
					if !errors.Is(result, tc.cause) {
						t.Errorf("%s: errors.Is(%v, %v) = false", key, result, tc.cause)
					}
				}
				if !found {
					t.Errorf("no validation error for %s in %v", key, vr)
				}
			}
		})
	}
}
