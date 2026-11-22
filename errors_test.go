package publiccode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/publiccodeyml/libpubliccode/v5/internal/httpclient"
)

func TestParseErrorError(t *testing.T) {
	e := ParseError{Reason: "something went wrong"}
	if e.Error() != "something went wrong" {
		t.Errorf("unexpected: %q", e.Error())
	}
}

func TestValidationErrorErrorEmptyKey(t *testing.T) {
	e := ValidationError{Key: "", Description: "bad field", Line: 1, Column: 2}
	got := e.Error()
	want := "publiccode.yml:1:2: error: bad field"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestValidationErrorErrorWithKey(t *testing.T) {
	e := ValidationError{Key: "foo", Description: "bad field", Line: 3, Column: 4}
	got := e.Error()
	want := "publiccode.yml:3:4: error: foo: bad field"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestValidationErrorErrorWithFile(t *testing.T) {
	e := ValidationError{File: "my-amazing-code.yaml", Key: "foo", Description: "bad field", Line: 3, Column: 4}
	got := e.Error()
	want := "my-amazing-code.yaml:3:4: error: foo: bad field"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestValidationWarningErrorWithFile(t *testing.T) {
	e := ValidationWarning{File: "my-amazing-code.yaml", Key: "foo", Description: "minor issue", Line: 5, Column: 6}
	got := e.Error()
	want := "my-amazing-code.yaml:5:6: warning: foo: minor issue"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestValidationErrorMarshalJSON(t *testing.T) {
	e := ValidationError{Key: "foo", Description: "bad", Line: 1, Column: 1}
	b, err := e.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}

	if m["type"] != "error" {
		t.Errorf("expected type=error, got %v", m["type"])
	}
	if m["key"] != "foo" {
		t.Errorf("expected key=foo, got %v", m["key"])
	}
}

func TestValidationWarningMarshalJSON(t *testing.T) {
	e := ValidationWarning{Key: "foo", Description: "minor", Line: 2, Column: 3}
	b, err := e.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}

	if m["type"] != "warning" {
		t.Errorf("expected type=warning, got %v", m["type"])
	}
}

func TestValidationErrorMarshalJSONFile(t *testing.T) {
	e := ValidationError{File: "my-amazing-code.yaml", Key: "foo", Description: "bad", Line: 1, Column: 2}
	b, err := e.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	got := string(b)
	want := `{"file":"my-amazing-code.yaml","key":"foo","description":"bad","line":1,"column":2,"type":"error"}`
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestValidationWarningMarshalJSONFile(t *testing.T) {
	e := ValidationWarning{File: "my-amazing-code.yaml", Key: "foo", Description: "minor", Line: 1, Column: 2}
	b, err := e.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	got := string(b)
	want := `{"file":"my-amazing-code.yaml","key":"foo","description":"minor","line":1,"column":2,"type":"warning"}`
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestValidationWarningErrorEmptyKey(t *testing.T) {
	e := ValidationWarning{Key: "", Description: "minor issue", Line: 5, Column: 6}
	got := e.Error()
	want := "publiccode.yml:5:6: warning: minor issue"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNewUnreachableErrorNamesURLOnce(t *testing.T) {
	u, _ := url.Parse("https://example.invalid/x/y.git")
	got := newUnreachableError("url", (*URL)(u), errors.New("no such host"))

	want := "'https://example.invalid/x/y.git' not reachable: no such host"
	if got.Description != want {
		t.Errorf("Description = %q, want %q", got.Description, want)
	}
}

func TestNewNotImageErrorNamesURLOnce(t *testing.T) {
	const raw = "https://example.com/raw/shot.png"

	tests := []struct {
		value string
		want  string
	}{
		{raw, "'" + raw + "' is not an image: not found"},
		{"shot.png", "'shot.png' is not an image: downloading " + raw + ": not found"},
	}

	for _, tt := range tests {
		err := &downloadError{URL: raw, Err: errors.New("not found")}

		got := newNotImageError("screenshots[0]", tt.value, err)
		if got.Description != tt.want {
			t.Errorf("Description = %q, want %q", got.Description, tt.want)
		}
	}
}

func TestValidationResultsHasErrors(t *testing.T) {
	warning := ValidationWarning{Key: "foo", Description: "minor warning"}
	validationError := ValidationError{Key: "bar", Description: "bad field"}

	tests := []struct {
		name    string
		results ValidationResults
		want    bool
	}{
		{"nil", nil, false},
		{"only warnings", ValidationResults{warning, warning}, false},
		{"one error", ValidationResults{validationError}, true},
		{"error after a warning", ValidationResults{warning, validationError}, true},
		{"wrapped error", ValidationResults{fmt.Errorf("wrapped: %w", validationError)}, true},
		{"other error type", ValidationResults{errors.New("plain")}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.results.HasErrors(); got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNetworkCause(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want error
	}{
		{"rate limited", fmt.Errorf("downloading: %w", httpclient.ErrRateLimited), ErrRateLimited},
		{"deadline", fmt.Errorf("requesting: %w", context.DeadlineExceeded), ErrTimeout},
		{"net timeout", &net.DNSError{Err: "i/o timeout", IsTimeout: true}, ErrTimeout},
		{"not found", fmt.Errorf("downloading: %w", httpclient.ErrNotFound), nil},
		{"refused", &net.OpError{Op: "dial", Err: errors.New("connection refused")}, nil},
		{"nil", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := newValidationError("url", "not reachable").withCause(tc.err).Unwrap(); !errors.Is(got, tc.want) {
				t.Errorf("Unwrap() = %v, want %v", got, tc.want)
			}
		})
	}
}
