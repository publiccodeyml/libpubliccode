package publiccode

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"
)

type testType struct {
	file string
	err  error
}

var cwd, cwdErr = os.Getwd()

func testWorkingDir(t *testing.T) string {
	t.Helper()
	if cwdErr != nil {
		t.Fatalf("failed to get cwd: %v", cwdErr)
	}

	return cwd
}

// Parse the YAML file passed as argument, using the current directory
// as base path.
//
// Return nil if the parsing succeeded or an error if it failed.
func parse(file string) error {
	var p *Parser
	var err error

	if p, err = NewDefaultParser(); err != nil {
		return err
	}

	_, err = p.Parse(file)

	return err
}

func parseNoNetwork(file string) error {
	var p *Parser
	var err error

	if p, err = NewParser(ParserConfig{DisableNetwork: true}); err != nil {
		return err
	}

	_, err = p.Parse(file)

	return err
}

// Check all the YAML files matching the glob pattern and fail for each file
// with parsing or validation errors.
func checkValidFiles(t *testing.T, pattern string) {
	t.Helper()
	testFiles, _ := filepath.Glob(pattern)
	for _, file := range testFiles {
		t.Run(file, func(t *testing.T) {
			if err := parse(file); err != nil {
				t.Errorf("[%s] validation failed for valid file: %T - %s\n", file, err, err)
			}
		})
	}
}

// Check all the YAML files matching the glob pattern and fail for each file
// with parsing or validation errors, with the network disabled.
func checkValidFilesNoNetwork(t *testing.T, pattern string) {
	t.Helper()
	testFiles, _ := filepath.Glob(pattern)
	for _, file := range testFiles {
		t.Run(file, func(t *testing.T) {
			if err := parseNoNetwork(file); err != nil {
				t.Errorf("[%s] validation failed for valid file: %T - %s\n", file, err, err)
			}
		})
	}
}

// Return a copy of the expected diagnostics carrying the name of the file
// they refer to.
func withFileName(name string, expected error) error {
	var results ValidationResults
	ok := errors.As(expected, &results)
	if !ok {
		return expected
	}

	stamped := make(ValidationResults, len(results))
	copy(stamped, results)

	setFileName(stamped, name)

	return stamped
}

// The expected diagnostics are written with no file name, because every test
// here parses a file whose name the parser reports on each of them.
func checkParseErrors(t *testing.T, err error, test testType) {
	t.Helper()
	expected := withFileName(path.Base(test.file), test.err)

	switch {
	case expected == nil && err != nil:
		t.Errorf("[%s] unexpected error: %v\n", test.file, err)
	case expected != nil && err == nil:
		t.Errorf("[%s] no error generated\n", test.file)
	case expected != nil && err != nil:
		if !reflect.DeepEqual(expected, err) {
			t.Errorf("[%s] wrong error generated:\n%T - %s\n- instead of:\n%T - %s", test.file, err, err, expected, expected)
		}
	}
}
