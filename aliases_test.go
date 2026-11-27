package publiccode

import (
	"fmt"
	"strings"
	"testing"

	"github.com/goccy/go-yaml/parser"
)

// sequenceOf returns a flow sequence of n scalars.
func sequenceOf(n int) string {
	return "[" + strings.TrimSuffix(strings.Repeat("x, ", n), ", ") + "]"
}

func TestFindAliasOverflow(t *testing.T) {
	// Each level aliases the one before three times.
	var nested strings.Builder

	nested.WriteString("l0: &l0 [a, b, c]\n")

	for i := 1; i < 60; i++ {
		fmt.Fprintf(&nested, "l%d: &l%d [*l%d, *l%d, *l%d]\n", i, i, i-1, i-1, i-1)
	}

	nested.WriteString("last: *l59\n")

	tests := []struct {
		name string
		yaml string
		// line of the alias reported, 0 for none
		line int
	}{
		{"no aliases", "a: [x, y]\nb: {c: d}\n", 0},
		{
			// The sequence and its entries make maxAliasNodes nodes.
			"at the limit",
			"a: &a " + sequenceOf(maxAliasNodes-1) + "\nb: *a\n",
			0,
		},
		{
			"past the limit",
			"a: &a " + sequenceOf(maxAliasNodes-1) + "\nb: *a\ns: &s x\nc: *s\n",
			4,
		},
		{
			"merge keys",
			"a: &a {k: " + sequenceOf(maxAliasNodes/2) + "}\nb: {<<: *a}\nc: {<<: *a}\n",
			3,
		},
		{
			"latest anchor",
			"a: &a " + sequenceOf(maxAliasNodes) + "\nb: &a x\nc: *a\nd: *a\n",
			0,
		},
		{"alias of its own anchor", "a: &a [x, *a]\nb: &b {<<: *b}\n", 0},
		{"unknown anchor", "a: *nope\n", 0},
		{"nested", nested.String(), 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := parser.ParseBytes([]byte(tt.yaml), 0)
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			line := 0
			if alias := findAliasOverflow(file.Docs[0].Body); alias != nil {
				line = alias.GetToken().Position.Line
			}

			if line != tt.line {
				t.Errorf("got alias at line %d, want %d", line, tt.line)
			}
		})
	}
}
