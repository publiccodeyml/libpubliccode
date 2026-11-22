package publiccode

import (
	"errors"
	"slices"
	"testing"
)

func TestDeprecatedLicenseWarnings(t *testing.T) {
	tests := []struct {
		expression string
		want       []string
	}{
		{"AGPL-3.0-or-later", nil},
		{"GPL-3.0+", nil},
		{"MIT OR Apache-2.0", nil},
		{"Invalid License", nil},
		{"", nil},
		{"gpl-3.0", []string{"'GPL-3.0' is a DEPRECATED SPDX license identifier. Use 'GPL-3.0-only' or 'GPL-3.0-or-later' instead"}},
		{"LGPL-2.1 OR LGPL-2.1", []string{"'LGPL-2.1' is a DEPRECATED SPDX license identifier. Use 'LGPL-2.1-only' or 'LGPL-2.1-or-later' instead"}},
		{"GPL-2.0 WITH Classpath-exception-2.0", []string{"'GPL-2.0' is a DEPRECATED SPDX license identifier. Use 'GPL-2.0-only' or 'GPL-2.0-or-later' instead"}},
		{"GPL-2.0-with-classpath-exception", []string{"'GPL-2.0-with-classpath-exception' is a DEPRECATED SPDX license identifier. Use its replacement from https://spdx.org/licenses instead"}},
	}

	for _, test := range tests {
		t.Run(test.expression, func(t *testing.T) {
			var got []string

			for _, result := range deprecatedLicenseWarnings("legal.license", test.expression) {
				warning, ok := errors.AsType[ValidationWarning](result)
				if !ok || warning.Key != "legal.license" {
					t.Fatalf("got %#v, want a legal.license ValidationWarning", result)
				}

				got = append(got, warning.Description)
			}

			if !slices.Equal(got, test.want) {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}
