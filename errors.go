package publiccode

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
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
}

func (e ValidationError) Error() string {
	key := ""
	if e.Key != "" {
		key = e.Key + ": "
	}

	return fmt.Sprintf("%s:%d:%d: error: %s%s", e.fileName(), e.Line, e.Column, key, e.Description)
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
// it contains only warnings. Inspect each element to determine its severity.
// ValidationResults does not unwrap its elements for errors.Is or errors.As.
type ValidationResults []error //nolint:errname // intentionally named as a collection, not an error type

func (vr ValidationResults) Error() string {
	s := make([]string, 0, len(vr))
	for _, e := range vr {
		s = append(s, e.Error())
	}

	return strings.Join(s, "\n")
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
