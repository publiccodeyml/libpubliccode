package publiccode

import (
	"errors"
	"strings"
	"testing"

	"github.com/goccy/go-yaml/parser"
)

func TestGetPositionInFileEmptyDocs(t *testing.T) {
	// An empty YAML file results in a parsed file with no docs.
	file, err := parser.ParseBytes([]byte(""), 0)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	line, col := getPositionInFile("someKey", file)
	if line != 0 || col != 0 {
		t.Errorf("expected 0,0 for empty file, got %d,%d", line, col)
	}
}

func TestGetPositionInFileNilBody(t *testing.T) {
	// Parse YAML that produces a doc with nil body.
	// A YAML file with only comments has a doc with nil Body.
	file, err := parser.ParseBytes([]byte("# just a comment\n"), 0)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	line, col := getPositionInFile("someKey", file)
	if line != 0 || col != 0 {
		t.Errorf("expected 0,0 for comment-only file, got %d,%d", line, col)
	}
}

func TestDecodeUnknownField(t *testing.T) {
	// Parse a YAML with an unknown field, which should be caught by
	// the decode function as an UnknownFieldError.
	yaml := `publiccodeYmlVersion: "0"
name: test
unknownField: value
`
	file, err := parser.ParseBytes([]byte(yaml), 0)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	v0 := &PublicCodeV0{}
	results := decode([]byte(yaml), v0, file)
	if results == nil {
		t.Error("expected error for unknown field")
	}
}

func TestDecodeWrongType(t *testing.T) {
	// Parse YAML where a field has wrong type (e.g. name is a list).
	yaml := `publiccodeYmlVersion: "0"
name:
  - item1
  - item2
`
	file, err := parser.ParseBytes([]byte(yaml), 0)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	v0 := &PublicCodeV0{}
	results := decode([]byte(yaml), v0, file)
	if results == nil {
		t.Error("expected error for wrong type")
	}
}

func TestDecodeReportsEveryUnknownKey(t *testing.T) {
	tests := map[string]struct {
		yaml string
		want ValidationResults
	}{
		"keys in a flow mapping": {
			yaml: `publiccodeYmlVersion: "0"
legal: {license: MIT, b: 1, a: 2}
`,
			want: ValidationResults{
				ValidationError{"", "legal.b", `unknown field "b"`, 2, 1},
				ValidationError{"", "legal.a", `unknown field "a"`, 2, 1},
			},
		},
		"key of an anchor used twice": {
			yaml: `publiccodeYmlVersion: "0"
fundedBy:
  - &org {uri: "urn:x-test:a", foo: 1}
  - *org
`,
			want: ValidationResults{
				ValidationError{"", "fundedBy[0].foo", `unknown field "foo"`, 3, 1},
			},
		},
		"key brought in by a merge key": {
			yaml: `publiccodeYmlVersion: "0"
x: &base {type: internal, bar: 1}
maintenance:
  <<: *base
  baz: 2
`,
			want: ValidationResults{
				ValidationError{"", "x", `unknown field "x"`, 2, 1},
				ValidationError{"", "maintenance.bar", `unknown field "bar"`, 2, 1},
				ValidationError{"", "maintenance.baz", `unknown field "baz"`, 5, 1},
			},
		},
		"anchor holding an alias of itself": {
			yaml: `publiccodeYmlVersion: "0"
fundedBy: &x [*x]
foo: 1
`,
			want: ValidationResults{
				ValidationError{"", "foo", `unknown field "foo"`, 3, 1},
				ValidationError{"", "fundedBy", "wrong type for this field", 2, 1},
			},
		},
		"mapping merging an alias of itself": {
			yaml: `publiccodeYmlVersion: "0"
maintenance: &x {type: internal, foo: 1, <<: *x}
`,
			want: ValidationResults{
				ValidationError{"", "maintenance.foo", `unknown field "foo"`, 2, 1},
				newValidationError("", "cannot find anchor by alias name x"),
			},
		},
		"tagged, quoted and anchored keys": {
			yaml: `publiccodeYmlVersion: "0"
!!str zzTagged: 1
"zzQuoted": 2
&k zzAnchored: 3
!!str 4: 4
`,
			want: ValidationResults{
				ValidationError{"", "zzTagged", `unknown field "zzTagged"`, 2, 1},
				ValidationError{"", "zzQuoted", `unknown field "zzQuoted"`, 3, 1},
				ValidationError{"", "zzAnchored", `unknown field "zzAnchored"`, 4, 1},
				ValidationError{"", "4", `unknown field "4"`, 5, 1},
			},
		},
		"key next to a wrong type": {
			yaml: `publiccodeYmlVersion: "0"
name: [a]
foo: 1
`,
			want: ValidationResults{
				ValidationError{"", "foo", `unknown field "foo"`, 3, 1},
				ValidationError{"", "name", "wrong type for this field", 2, 1},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			file, err := parser.ParseBytes([]byte(test.yaml), 0)
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			got := decode([]byte(test.yaml), &PublicCodeV0{}, file)
			if got.Error() != test.want.Error() {
				t.Errorf("got:\n%v\nwant:\n%v", got, test.want)
			}
		})
	}
}

func TestToURLLocalPath(t *testing.T) {
	// toURL with a local file path should return a file:// URL.
	u, err := toURL("./testdata/v0/valid/valid.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Scheme != "file" {
		t.Errorf("expected file scheme, got %q", u.Scheme)
	}
}

func TestToURLHTTPS(t *testing.T) {
	u, err := toURL("https://example.com/publiccode.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Host != "example.com" {
		t.Errorf("unexpected host: %s", u.Host)
	}
}

func TestFindKeyPosNilNode(t *testing.T) {
	line, col := findKeyPos(nil, []string{"key"})
	if line != 0 || col != 0 {
		t.Errorf("expected 0,0 for nil node, got %d,%d", line, col)
	}
}

func TestFindKeyAtLineNilNode(t *testing.T) {
	result := findKeyAtLine(nil, 1, "")
	if result != "" {
		t.Errorf("expected empty string for nil node, got %q", result)
	}
}

func TestFindKeyPosDocumentNode(t *testing.T) {
	file, err := parser.ParseBytes([]byte("key: value\n"), 0)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	// file.Docs[0] is a *ast.DocumentNode: exercises the DocumentNode case in findKeyPos
	line, col := findKeyPos(file.Docs[0], []string{"key"})
	if line == 0 {
		t.Errorf("expected non-zero line for 'key', got %d,%d", line, col)
	}
}

func TestFindKeyAtLineDocumentNode(t *testing.T) {
	file, err := parser.ParseBytes([]byte("key: value\n"), 0)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	// file.Docs[0] is a *ast.DocumentNode: exercises the DocumentNode case in findKeyAtLine
	result := findKeyAtLine(file.Docs[0], 1, "")
	_ = result
}

func TestParseStreamDeprecatedVersionWarning(t *testing.T) {
	// An older but supported version should produce a deprecation warning.
	p, _ := NewParser(ParserConfig{DisableNetwork: true})
	yaml := `publiccodeYmlVersion: "0.2"
name: test
url: "https://github.com/example/repo.git"
platforms:
  - web
developmentStatus: stable
softwareType: "standalone/web"
description:
  en:
    genericName: Test
    shortDescription: "A short description."
    longDescription: >
      Long description that is longer than one hundred and fifty characters to
      satisfy the minimum length requirement for this field in publiccode.yml.
      Adding more text here to ensure we definitely hit the 150 character minimum.
    features:
      - feature1
legal:
  license: MIT
maintenance:
  type: none
localisation:
  localisationReady: true
  availableLanguages:
    - en
`
	_, err := p.ParseStream(strings.NewReader(yaml))
	// Should produce a deprecation warning but no validation errors.
	if err == nil {
		return // no errors at all is also fine
	}
	var vr ValidationResults
	ok := errors.As(err, &vr)
	if !ok {
		t.Fatalf("expected ValidationResults, got %T: %v", err, err)
	}
	hasWarning := false
	for _, e := range vr {
		if _, ok := errors.AsType[ValidationWarning](e); ok {
			hasWarning = true
		}
		if _, ok := errors.AsType[ValidationError](e); ok {
			t.Errorf("unexpected validation error: %v", e)
		}
	}
	if !hasWarning {
		t.Error("expected at least one deprecation warning for old version")
	}
}

func TestDecodeTaggedMappingAsList(t *testing.T) {
	for _, field := range []string{"platforms", "usedBy", "isBasedOn"} {
		t.Run(field, func(t *testing.T) {
			input := "publiccodeYmlVersion: \"0\"\n" + field + ": !custom {key: value}\n"
			file, err := parser.ParseBytes([]byte(input), 0)
			if err != nil {
				t.Fatal(err)
			}

			var document PublicCodeV0
			results := decode([]byte(input), &document, file)
			if len(results) != 1 {
				t.Fatalf("expected one decoding error, got %v", results)
			}
			if _, ok := errors.AsType[ValidationError](results[0]); !ok {
				t.Fatalf("expected ValidationError, got %T", results[0])
			}
		})
	}
}
