package httpclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

type step struct {
	status     int
	retryAfter string
	rateReset  string
	remaining  string
}

// newClient starts a server that answers the steps in order and then 200
// with a body. The client records each wait instead of sleeping.
func newClient(t *testing.T, steps ...step) (*Client, string, *[]time.Duration) {
	t.Helper()

	var waits []time.Duration
	calls := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls < len(steps) {
			s := steps[calls]
			calls++
			if s.retryAfter != "" {
				w.Header().Set("Retry-After", s.retryAfter)
			}
			if s.rateReset != "" {
				w.Header().Set("X-Ratelimit-Reset", s.rateReset)
			}
			if s.remaining != "" {
				w.Header().Set("X-Ratelimit-Remaining", s.remaining)
			}
			w.WriteHeader(s.status)

			return
		}
		_, _ = w.Write([]byte("body"))
	}))
	t.Cleanup(srv.Close)

	c := New(srv.Client())
	c.sleep = func(d time.Duration) { waits = append(waits, d) }
	c.now = func() time.Time { return time.Unix(1000, 0) }

	return c, srv.URL + "/file.txt", &waits
}

func TestGetReturnsTheBody(t *testing.T) {
	c, url, waits := newClient(t)

	body, err := c.Get(url, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "body" {
		t.Errorf("body = %q, want %q", body, "body")
	}
	if len(*waits) != 0 {
		t.Errorf("waited %v, want no wait", *waits)
	}
}

func TestGetSendsTheHeaders(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
	}))
	t.Cleanup(srv.Close)

	_, err := New(srv.Client()).Get(srv.URL, map[string]string{"Authorization": "token x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "token x" {
		t.Errorf("Authorization = %q, want %q", got, "token x")
	}
}

func TestGetNotFound(t *testing.T) {
	c, url, _ := newClient(t, step{status: http.StatusNotFound})

	_, err := c.Get(url, nil)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if err.Error() != "not found" {
		t.Errorf("err = %q, want %q", err, "not found")
	}
}

func TestGetForbiddenIsNotRetried(t *testing.T) {
	c, url, waits := newClient(t, step{status: http.StatusForbidden})

	_, err := c.Get(url, nil)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("err = %v, want ErrForbidden", err)
	}
	if len(*waits) != 0 {
		t.Errorf("waited %v, want no wait", *waits)
	}
}

func TestGetForbiddenWithRetryAfterIsRetried(t *testing.T) {
	c, url, waits := newClient(t, step{status: http.StatusForbidden, retryAfter: "2"})

	body, err := c.Get(url, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "body" {
		t.Errorf("body = %q, want %q", body, "body")
	}
	if want := []time.Duration{2 * time.Second}; !slices.Equal(*waits, want) {
		t.Errorf("waits = %v, want %v", *waits, want)
	}
}

func TestGetHonorsRetryAfterOnRateLimit(t *testing.T) {
	c, url, waits := newClient(t,
		step{status: http.StatusTooManyRequests, retryAfter: "3"},
		step{status: http.StatusTooManyRequests, retryAfter: "5"},
	)

	body, err := c.Get(url, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "body" {
		t.Errorf("body = %q, want %q", body, "body")
	}
	if want := []time.Duration{3 * time.Second, 5 * time.Second}; !slices.Equal(*waits, want) {
		t.Errorf("waits = %v, want %v", *waits, want)
	}
}

func TestGetBacksOffWithoutRetryAfter(t *testing.T) {
	c, url, waits := newClient(t,
		step{status: http.StatusTooManyRequests},
		step{status: http.StatusTooManyRequests, retryAfter: "garbage"},
		step{status: http.StatusTooManyRequests},
	)

	_, err := c.Get(url, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute}; !slices.Equal(*waits, want) {
		t.Errorf("waits = %v, want %v", *waits, want)
	}
}

func TestGetGivesUpAfterMaxAttempts(t *testing.T) {
	steps := make([]step, maxAttempts+1)
	for i := range steps {
		steps[i] = step{status: http.StatusTooManyRequests, retryAfter: "1"}
	}
	c, url, waits := newClient(t, steps...)

	_, err := c.Get(url, nil)
	if err == nil {
		t.Fatal("expected an error after the last attempt")
	}
	if len(*waits) != maxAttempts-1 {
		t.Errorf("waited %d times, want %d", len(*waits), maxAttempts-1)
	}
}

func TestGetServerErrorIsNotRetried(t *testing.T) {
	c, url, waits := newClient(t, step{status: http.StatusInternalServerError})

	_, err := c.Get(url, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if len(*waits) != 0 {
		t.Errorf("waited %v, want no wait", *waits)
	}
}

func TestGetTransportError(t *testing.T) {
	_, err := New(&http.Client{}).Get("http://localhost:1/", nil)
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestGetInvalidURL(t *testing.T) {
	_, err := New(nil).Get("hktp://bad url", nil)
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestNewDefaultsToATimeout(t *testing.T) {
	c := New(nil)

	hc, ok := c.doer.(*http.Client)
	if !ok {
		t.Fatalf("doer is %T, want *http.Client", c.doer)
	}
	if hc.Timeout == 0 {
		t.Error("default client has no timeout")
	}
}

func TestGetForbiddenWaitsForTheRateLimitReset(t *testing.T) {
	c, url, waits := newClient(t, step{status: http.StatusForbidden, rateReset: "1030", remaining: "0"})

	body, err := c.Get(url, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "body" {
		t.Errorf("body = %q, want %q", body, "body")
	}
	if want := []time.Duration{30 * time.Second}; !slices.Equal(*waits, want) {
		t.Errorf("waits = %v, want %v", *waits, want)
	}
}

func TestGetForbiddenWithQuotaLeftIsNotRetried(t *testing.T) {
	c, url, waits := newClient(t, step{status: http.StatusForbidden, rateReset: "1030", remaining: "12"})

	_, err := c.Get(url, nil)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("err = %v, want ErrForbidden", err)
	}
	if len(*waits) != 0 {
		t.Errorf("waited %v, want no wait", *waits)
	}
}

func TestGetRateLimitResetInThePastRetriesAtOnce(t *testing.T) {
	c, url, waits := newClient(t, step{status: http.StatusTooManyRequests, rateReset: "900"})

	if _, err := c.Get(url, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []time.Duration{0}; !slices.Equal(*waits, want) {
		t.Errorf("waits = %v, want %v", *waits, want)
	}
}

func TestGetRetryAfterWinsOverTheRateLimitReset(t *testing.T) {
	c, url, waits := newClient(t, step{status: http.StatusTooManyRequests, retryAfter: "3", rateReset: "1030"})

	if _, err := c.Get(url, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []time.Duration{3 * time.Second}; !slices.Equal(*waits, want) {
		t.Errorf("waits = %v, want %v", *waits, want)
	}
}

func TestGetGivesUpSoonerWithoutAWaitFromTheServer(t *testing.T) {
	steps := make([]step, maxBlindAttempts+1)
	for i := range steps {
		steps[i] = step{status: http.StatusTooManyRequests}
	}
	c, url, waits := newClient(t, steps...)

	_, err := c.Get(url, nil)
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("err = %v, want ErrRateLimited", err)
	}
	if want := []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute}; !slices.Equal(*waits, want) {
		t.Errorf("waits = %v, want %v", *waits, want)
	}
}
