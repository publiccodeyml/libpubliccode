package publiccode

import (
	"slices"
	"strings"

	"github.com/github/go-spdx/v2/spdxexp"
	"github.com/github/go-spdx/v2/spdxexp/spdxlicenses"
)

// deprecatedLicenseWarnings warns once for every license identifier in the
// SPDX expression that the SPDX License List marks as deprecated. These
// identifiers are still valid, so the manifest is too, but some of them,
// like GPL-3.0, don't say whether later versions of the license apply.
func deprecatedLicenseWarnings(key string, expression string) ValidationResults {
	// An invalid expression gives no licenses here: is_spdx_expression
	// already reports it.
	licenses, _ := spdxexp.ExtractLicenses(expression)

	var deprecated []string

	for _, license := range licenses {
		// A license with an exception comes back as "<id> WITH <exception>".
		id, _, _ := strings.Cut(license, " WITH ")

		if isDeprecated, canonical := spdxlicenses.IsDeprecatedLicense(id); isDeprecated &&
			!slices.Contains(deprecated, canonical) {
			deprecated = append(deprecated, canonical)
		}
	}

	slices.Sort(deprecated)

	var vr ValidationResults

	for _, id := range deprecated {
		only, orLater := id+"-only", id+"-or-later"

		if valid, _ := spdxexp.ValidateLicenses([]string{only, orLater}); valid {
			vr = append(vr, newValidationWarningf(
				key,
				"'%s' is a DEPRECATED SPDX license identifier. Use '%s' or '%s' instead",
				id, only, orLater,
			))

			continue
		}

		vr = append(vr, newValidationWarningf(
			key,
			"'%s' is a DEPRECATED SPDX license identifier. Use its replacement from https://spdx.org/licenses instead",
			id,
		))
	}

	return vr
}
