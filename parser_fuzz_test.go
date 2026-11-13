package publiccode

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func FuzzParseStream(f *testing.F) {
	files, err := filepath.Glob("testdata/*/*/*.yml")
	if err != nil {
		f.Fatal(err)
	}

	if len(files) == 0 {
		f.Fatal("no manifest fixtures found")
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			f.Fatal(err)
		}

		f.Add(data)
	}

	parser, err := NewParser(ParserConfig{DisableExternalChecks: true})
	if err != nil {
		f.Fatal(err)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Invalid manifests may return errors, but must never panic.
		_, _ = parser.ParseStream(bytes.NewReader(data))
	})
}
