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
	sleep func(time.Duration)
	now   func() time.Time
}

var (
	ErrNotFound         = errors.New("not found")
	ErrForbidden        = errors.New("forbidden")
	ErrRateLimited      = errors.New("rate limited")
	ErrUnexpectedStatus = errors.New("unexpected status")
)

// maxAttempts caps the retries when the server says how long to wait, so
// a server that always answers 429 does not block a validation forever.
const maxAttempts = 8

// maxBlindAttempts caps the retries when the server gives no time. The
// waits start at one minute, the floor GitHub asks for on a secondary
// rate limit, and double each time: seven minutes in total.
const maxBlindAttempts = 4

// New returns a Client that sends its requests through doer. When doer is
// nil it uses a default client with a one minute timeout.
func New(doer Doer) *Client {
	if doer == nil {
		doer = &http.Client{Timeout: 60 * time.Second}
	}

	return &Client{doer: doer, sleep: time.Sleep, now: time.Now}
}

// Get returns the body at url. On a 429, or on a 403 that is a rate limit,
// it waits and tries again: for as long as Retry-After says, until the
// time in X-Ratelimit-Reset, or with an exponential backoff from one
// minute when the server gave no time.
func (c *Client) Get(url string, headers map[string]string) ([]byte, error) {
	for attempt := 1; ; attempt++ {
		resp, err := c.do(url, headers)
		if err != nil {
			return nil, err
		}

		wait, told, retry := c.retryAfter(resp)
		if !retry {
			return read(resp)
		}

		_ = resp.Body.Close()

		limit := maxAttempts
		if !told {
			limit = maxBlindAttempts
			wait = time.Minute << (attempt - 1)
		}

		if attempt == limit {
			return nil, fmt.Errorf("%w after %d attempts: %s", ErrRateLimited, attempt, resp.Status)
		}

		c.sleep(wait)
	}
}

func (c *Client) do(url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
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
	retryAfter := resp.Header.Get("Retry-After")
	reset := resp.Header.Get("X-Ratelimit-Reset")

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
	case http.StatusForbidden:
		quotaUsedUp := reset != "" && resp.Header.Get("X-Ratelimit-Remaining") == "0"
		if retryAfter == "" && !quotaUsedUp {
			return 0, false, false
		}
	default:
		return 0, false, false
	}

	if secs, err := strconv.Atoi(retryAfter); err == nil {
		return time.Duration(secs) * time.Second, true, true
	}

	if epoch, err := strconv.ParseInt(reset, 10, 64); err == nil {
		return max(time.Unix(epoch, 0).Sub(c.now()), 0), true, true
	}

	return 0, false, true
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
