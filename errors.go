package publiccode

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/publiccodeyml/libpubliccode/v5/internal/httpclient"
)

// Causes of a failed network check. A [ValidationError] that reports an
// unreachable URL, logo or screenshot unwraps to one of them when the check
// failed for a reason that says nothing about the value, so callers can
// tell it apart with errors.Is and try the file again later.
// [ValidationResults] does not unwrap, so check each of its elements.
var (
	// ErrRateLimited means the server kept rate limiting the request, with
	// a 429 or a rate limit 403, after the parser's retries.
	ErrRateLimited = errors.New("rate limited")
	// ErrTimeout means the request did not complete within the parser's
	// timeout or the caller's deadline.
	ErrTimeout = errors.New("timed out")
)

// ParseError is generic parse error.
type ParseError struct {
	Reason string
}

func (e ParseError) Error() string {
	return e.Reason
}

// defaultFileName is reported by diagnostics coming from an input with no
// name of its own, as happens when parsing a stream.
const defaultFileName = "publiccode.yml"

// ValidationError reports a parsing or validation failure in a publiccode.yml file.
type ValidationError struct {
	// Name of the validated file, with no directory part. It is empty when
	// the input is a stream, which has no name.
	File        string `json:"file"`
	Key         string `json:"key"`
	Description string `json:"description"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`

	// cause is ErrRateLimited, ErrTimeout or nil.
	cause error
}

func (e ValidationError) Error() string {
	key := ""
	if e.Key != "" {
		key = e.Key + ": "
	}

	return fmt.Sprintf("%s:%d:%d: error: %s%s", e.fileName(), e.Line, e.Column, key, e.Description)
}

// Unwrap returns [ErrRateLimited] or [ErrTimeout] when e reports a network
// check that failed for that reason, and nil otherwise.
func (e ValidationError) Unwrap() error {
	return e.cause
}

func (e ValidationError) MarshalJSON() ([]byte, error) {
	type Ve ValidationError

	return json.Marshal(&struct {
		*Ve

		Type string `json:"type"`
	}{
		Ve:   (*Ve)(&e),
		Type: "error",
	})
}

func (e ValidationError) fileName() string {
	if e.File == "" {
		return defaultFileName
	}

	return e.File
}

func newValidationError(key string, description string) ValidationError {
	return ValidationError{Key: key, Description: description}
}

func newValidationErrorf(key string, description string, args ...any) ValidationError {
	return newValidationError(key, fmt.Sprintf(description, args...))
}

// withCause records why the network check behind e failed, when err says
// it was rate limited or timed out.
func (e ValidationError) withCause(err error) ValidationError {
	e.cause = networkCause(err)

	return e
}

func networkCause(err error) error {
	if errors.Is(err, httpclient.ErrRateLimited) {
		return ErrRateLimited
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}

	if timeout, ok := errors.AsType[interface {
		error
		Timeout() bool
	}](err); ok && timeout.Timeout() {
		return ErrTimeout
	}

	return nil
}

// ValidationWarning reports a condition that does not make a publiccode.yml file
// invalid, such as use of a deprecated field. It implements error so it can be
// returned alongside [ValidationError] values in [ValidationResults].
//
//nolint:errname,lll // ValidationWarning is intentionally named as a warning, not an error, even though it implements error.
type ValidationWarning ValidationError

func newValidationWarning(key string, description string) ValidationWarning {
	return ValidationWarning{Key: key, Description: description}
}

func newValidationWarningf(key string, description string, args ...any) ValidationWarning {
	return newValidationWarning(key, fmt.Sprintf(description, args...))
}

func (e ValidationWarning) Error() string {
	key := ""
	if e.Key != "" {
		key = e.Key + ": "
	}

	return fmt.Sprintf("%s:%d:%d: warning: %s%s", ValidationError(e).fileName(), e.Line, e.Column, key, e.Description)
}

func (e ValidationWarning) MarshalJSON() ([]byte, error) {
	type Ve ValidationError

	return json.Marshal(&struct {
		*Ve

		Type string `json:"type"`
	}{
		Ve:   (*Ve)(&e),
		Type: "warning",
	})
}

// ValidationResults collects [ValidationError] and [ValidationWarning] values.
// [Parser.Parse] and [Parser.ParseStream] return it as a non-nil error even when
// it contains only warnings. Use [ValidationResults.HasErrors] to tell whether the
// file is valid, or inspect each element to determine its severity.
// ValidationResults does not unwrap its elements for errors.Is or errors.As.
type ValidationResults []error //nolint:errname // intentionally named as a collection, not an error type

func (vr ValidationResults) Error() string {
	s := make([]string, 0, len(vr))
	for _, e := range vr {
		s = append(s, e.Error())
	}

	return strings.Join(s, "\n")
}

// HasErrors reports whether vr contains a [ValidationError]. A file whose
// results hold only [ValidationWarning] values is valid.
func (vr ValidationResults) HasErrors() bool {
	return slices.ContainsFunc(vr, func(err error) bool {
		_, ok := errors.AsType[ValidationError](err)

		return ok
	})
}

// sortByPosition orders diagnostics as they appear in the file. Some checks
// walk maps such as description, whose iteration order changes from one run
// to the next. Diagnostics on the same position keep their relative order.
func sortByPosition(vr ValidationResults) {
	slices.SortStableFunc(vr, func(a, b error) int {
		aLine, aColumn := position(a)
		bLine, bColumn := position(b)

		return cmp.Or(cmp.Compare(aLine, bLine), cmp.Compare(aColumn, bColumn))
	})
}

func position(err error) (int, int) {
	if valErr, ok := errors.AsType[ValidationError](err); ok {
		return valErr.Line, valErr.Column
	}

	if valWarn, ok := errors.AsType[ValidationWarning](err); ok {
		return valWarn.Line, valWarn.Column
	}

	return 0, 0
}

// newUnreachableError reports that the URL in key did not answer. err
// gives the cause and does not name the URL, so the URL appears once.
func newUnreachableError(key string, u *URL, err error) ValidationError {
	return newValidationErrorf(key, "'%s' not reachable: %s", (*url.URL)(u).String(), err.Error()).withCause(err)
}

// newNotImageError reports that the file value in key is not an image.
// When the request failed for the URL value itself, the URL is named only
// once. A URL resolved from a relative path stays in the message.
func newNotImageError(key string, value string, err error) ValidationError {
	if dlErr, ok := errors.AsType[*downloadError](err); ok && dlErr.URL == value {
		err = dlErr.Err
	}

	return newValidationErrorf(key, "'%s' is not an image: %s", value, err.Error()).withCause(err)
}
