// Package httpclient downloads the URLs found in a publiccode.yml and
// retries when the server asks to slow down.
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Doer sends an HTTP request. *http.Client implements it.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client sends GET requests and retries the ones the server rate limits.
type Client struct {
	doer  Doer
	sleep func(context.Context, time.Duration) error
	now   func() time.Time
}

const (
	headerRetryAfter    = "Retry-After"
	headerRateReset     = "X-Ratelimit-Reset"
	headerRateRemaining = "X-Ratelimit-Remaining"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrForbidden        = errors.New("forbidden")
	ErrRateLimited      = errors.New("rate limited")
	ErrUnexpectedStatus = errors.New("unexpected status")

	errInvalidHeader = errors.New("invalid header")
)

// maxAttempts caps the retries when the server says how long to wait, so
// a server that always answers 429 does not block a validation forever.
const maxAttempts = 8

// maxBlindAttempts caps the retries when the server gives no time. The
// waits start at one minute, the floor GitHub asks for on a secondary
// rate limit, and double each time: seven minutes in total.
const maxBlindAttempts = 4

// maxRetryWait caps a wait the server asks for, so a Retry-After of a day
// or a far away X-Ratelimit-Reset does not stall a validation.
const maxRetryWait = 30 * time.Second

// New returns a Client that sends its requests through doer. When doer is
// nil it uses a default client with a one minute timeout.
func New(doer Doer) *Client {
	if doer == nil {
		doer = &http.Client{Timeout: 60 * time.Second}
	}

	return &Client{doer: doer, sleep: sleepWithContext, now: time.Now}
}

// Get returns the body at url. On a 429, or on a 403 that is a rate limit,
// it waits and tries again: for as long as Retry-After says, until the
// time in X-Ratelimit-Reset, or with an exponential backoff from one
// minute when the server gave no time. A wait the server asks for is
// capped at 30 seconds.
func (c *Client) Get(url string, headers map[string]string) ([]byte, error) {
	return c.GetWithContext(context.Background(), url, headers)
}

// GetWithContext is Get with a context. The request and the waits between
// retries stop when ctx is cancelled or reaches its deadline.
func (c *Client) GetWithContext(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	blind := 0

	for attempt := 1; ; attempt++ {
		resp, err := c.do(ctx, url, headers)
		if err != nil {
			return nil, err
		}

		wait, told, retry := c.retryAfter(resp)
		if !retry {
			return read(resp)
		}

		_ = resp.Body.Close()

		// The backoff doubles per blind retry, not per attempt: retries
		// where the server gave a time must not inflate it.
		if !told {
			blind++
			wait = time.Minute << (blind - 1)
		}

		if attempt >= maxAttempts || blind >= maxBlindAttempts {
			return nil, fmt.Errorf("%w after %d attempts: %s", ErrRateLimited, attempt, resp.Status)
		}

		if err := c.sleep(ctx, wait); err != nil {
			return nil, err
		}
	}
}

func (c *Client) do(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building the request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting %s: %w", url, err)
	}

	return resp, nil
}

// retryAfter says whether resp is a rate limit to retry and how long the
// server asked to wait. The second result is false when the server gave
// no time. A 403 is a rate limit when it carries Retry-After, or
// X-Ratelimit-Reset with no quota left, as GitHub answers when the API
// quota is used up. Any other 403 is a denial.
func (c *Client) retryAfter(resp *http.Response) (time.Duration, bool, bool) {
	retryAfter := resp.Header.Get(headerRetryAfter)
	reset := resp.Header.Get(headerRateReset)

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
	case http.StatusForbidden:
		quotaUsedUp := reset != "" && resp.Header.Get(headerRateRemaining) == "0"
		if retryAfter == "" && !quotaUsedUp {
			return 0, false, false
		}
	default:
		return 0, false, false
	}

	now := c.now()

	if wait, err := parseRetryAfter(retryAfter, now); err == nil {
		return wait, true, true
	}

	if wait, err := parseRateLimitReset(reset, now); err == nil {
		return wait, true, true
	}

	return 0, false, true
}

// parseRetryAfter reads Retry-After as delay-seconds or as an HTTP date
// (RFC 9110, section 10.2.3). ParseUint takes digits only, as the grammar
// does: no sign, so "-5" and "+5" are invalid. A number too large for
// uint64 is still valid and means the longest wait.
func parseRetryAfter(value string, now time.Time) (time.Duration, error) {
	seconds, err := strconv.ParseUint(value, 10, 64)
	if err == nil || errors.Is(err, strconv.ErrRange) {
		if seconds >= uint64(maxRetryWait/time.Second) {
			return maxRetryWait, nil
		}

		return time.Duration(seconds) * time.Second, nil
	}

	retryAt, err := http.ParseTime(value)
	if err != nil {
		return 0, fmt.Errorf("%w %s value %q", errInvalidHeader, headerRetryAfter, value)
	}

	if !retryAt.After(now) {
		return 0, nil
	}

	return capRetryWait(retryAt.Sub(now)), nil
}

// parseRateLimitReset returns the time left until the X-Ratelimit-Reset
// epoch, in seconds.
func parseRateLimitReset(value string, now time.Time) (time.Duration, error) {
	reset, err := strconv.ParseUint(value, 10, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0, fmt.Errorf("%w %s value %q", errInvalidHeader, headerRateReset, value)
	}

	nowSeconds := uint64(max(now.Unix(), 0))
	if reset <= nowSeconds {
		return 0, nil
	}

	waitSeconds := reset - nowSeconds
	if waitSeconds >= uint64(maxRetryWait/time.Second) {
		return maxRetryWait, nil
	}

	return time.Duration(waitSeconds) * time.Second, nil
}

func capRetryWait(wait time.Duration) time.Duration {
	if wait > maxRetryWait {
		return maxRetryWait
	}

	return wait
}

func sleepWithContext(ctx context.Context, wait time.Duration) error {
	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("waiting to retry: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func read(resp *http.Response) ([]byte, error) {
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, ErrNotFound
	case resp.StatusCode == http.StatusForbidden:
		return nil, ErrForbidden
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return nil, fmt.Errorf("%w %s", ErrUnexpectedStatus, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading the body: %w", err)
	}

	return body, nil
}
