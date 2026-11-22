package publiccode

import (
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"testing"
)

func TestValidTestcasesV0_NoNetwork(t *testing.T) {
	checkValidFilesNoNetwork(t, "testdata/v0/valid/no-network/*.yml")
}

func TestValidWithWarningTestcasesV0_NoNetwork(t *testing.T) {
	expected := map[string]error{
		"authorsFile.yml": ValidationResults{
			ValidationWarning{File: "", Key: "legal.authorsFile", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 63, Column: 3},
			ValidationWarning{File: "", Key: "IT", Description: "This key is DEPRECATED and will be removed in the future. Migrate its keys as their warnings say, then drop it", Line: 95, Column: 1},
		},
	}

	dir := "testdata/v0/valid_with_warnings/no-network/"
	testFiles, _ := filepath.Glob(dir + "*yml")
	for _, file := range testFiles {
		baseName := path.Base(file)
		if expected[baseName] == nil {
			t.Errorf("No expected data for file %s", baseName)
		}
		t.Run(file, func(t *testing.T) {
			err := parseNoNetwork(file)
			checkParseErrors(t, err, testType{file, expected[baseName]})
		})
	}
	for file := range expected {
		if !slices.Contains(testFiles, dir+file) {
			t.Errorf("No expected file %s", dir+file)
		}
	}
}

func TestInvalidTestcasesV0_NoNetwork(t *testing.T) {
	expected := map[string]error{
		// logo
		"logo_missing_file.yml": ValidationResults{
			ValidationError{File: "", Key: "logo", Description: "no such file: " + cwd + "/testdata/v0/invalid/no-network/no_such_file.png", Line: 18, Column: 1},
		},
		"logo_invalid_png.yml": ValidationResults{
			ValidationError{File: "", Key: "logo", Description: "image: unknown format", Line: 18, Column: 1},
		},

		// landingURL
		"landingURL_invalid.yml": ValidationResults{
			// Just a syntax check here, no check for reachability as network is disabled
			ValidationError{File: "", Key: "landingURL", Description: "landingURL must be an HTTP URL", Line: 8, Column: 1},
			ValidationWarning{File: "", Key: "usedBy", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 21, Column: 1},
			ValidationWarning{File: "", Key: "description.en.videos", Description: "This key is DEPRECATED and will be removed in the future. Link videos from the documentation or the website instead", Line: 62, Column: 5},
			ValidationWarning{File: "", Key: "dependsOn.open", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 102, Column: 3},
			ValidationWarning{File: "", Key: "IT", Description: "This key is DEPRECATED and will be removed in the future. Migrate its keys as their warnings say, then drop it", Line: 118, Column: 1},
		},

		// monochromeLogo
		"monochromeLogo_invalid_png.yml": ValidationResults{
			ValidationWarning{File: "", Key: "monochromeLogo", Description: "This key is DEPRECATED and will be removed in the future. Use 'logo' instead", Line: 18, Column: 1},
			ValidationError{File: "", Key: "monochromeLogo", Description: "image: unknown format", Line: 18, Column: 1},
		},

		// YAML 1.1 boolean aliases must be rejected
		"localisation_localisationReady_yaml11_yes.yml": ValidationResults{
			ValidationError{File: "", Key: "localisation.localisationReady", Description: "wrong type for this field", Line: 51, Column: 1},
			ValidationError{File: "", Key: "localisation.localisationReady", Description: "localisationReady is a required field", Line: 51, Column: 3},
			ValidationError{File: "", Key: "localisation.availableLanguages", Description: "availableLanguages is a required field", Line: 52, Column: 3},
		},
		"localisation_localisationReady_yaml11_no.yml": ValidationResults{
			ValidationError{File: "", Key: "localisation.localisationReady", Description: "wrong type for this field", Line: 51, Column: 1},
			ValidationError{File: "", Key: "localisation.localisationReady", Description: "localisationReady is a required field", Line: 51, Column: 3},
			ValidationError{File: "", Key: "localisation.availableLanguages", Description: "availableLanguages is a required field", Line: 52, Column: 3},
		},
		"localisation_localisationReady_yaml11_on.yml": ValidationResults{
			ValidationError{File: "", Key: "localisation.localisationReady", Description: "wrong type for this field", Line: 51, Column: 1},
			ValidationError{File: "", Key: "localisation.localisationReady", Description: "localisationReady is a required field", Line: 51, Column: 3},
			ValidationError{File: "", Key: "localisation.availableLanguages", Description: "availableLanguages is a required field", Line: 52, Column: 3},
		},
		"localisation_localisationReady_yaml11_off.yml": ValidationResults{
			ValidationError{File: "", Key: "localisation.localisationReady", Description: "wrong type for this field", Line: 51, Column: 1},
			ValidationError{File: "", Key: "localisation.localisationReady", Description: "localisationReady is a required field", Line: 51, Column: 3},
			ValidationError{File: "", Key: "localisation.availableLanguages", Description: "availableLanguages is a required field", Line: 52, Column: 3},
		},

		// supports
		"supports_unknown_alias.yml": ValidationResults{
			ValidationError{File: "", Key: "supports[0].id", Description: "id contains an unknown alias (see https://github.com/publiccodeyml/publiccode.yml/blob/main/docs/standard/aliases-list.rst)", Line: 17, Column: 5},
		},
	}

	dir := "testdata/v0/invalid/no-network/"
	testFiles, _ := filepath.Glob(dir + "*yml")
	for _, file := range testFiles {
		baseName := path.Base(file)
		if expected[baseName] == nil {
			t.Errorf("No expected data for file %s", baseName)
		}
		t.Run(file, func(t *testing.T) {
			err := parseNoNetwork(file)
			checkParseErrors(t, err, testType{file, expected[baseName]})
		})
	}
	for file := range expected {
		if !slices.Contains(testFiles, dir+file) {
			t.Errorf("No expected file %s", dir+file)
		}
	}
}

//nolint:maintidx // The table keeps all invalid v0 fixtures together.
func TestInvalidTestcasesV0(t *testing.T) {
	expected := map[string]error{
		// publiccodeYmlVersion
		"publiccodeYmlVersion_missing.yml": ValidationResults{ValidationError{File: "", Key: "publiccodeYmlVersion", Description: "publiccodeYmlVersion is a required field", Line: 0, Column: 0}},
		"publiccodeYmlVersion_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "publiccodeYmlVersion", Description: "unsupported version: '1'. Supported versions: 0, 0.2, 0.2.0, 0.2.1, 0.2.2, 0.3, 0.3.0, 0.4, 0.4.0, 0.5.0, 0.5, 0.6.0, 0.6, 0.7.0, 0.7, 0.8.0, 0.8", Line: 0, Column: 0},
		},
		"publiccodeYmlVersion_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "publiccodeYmlVersion", Description: "wrong type for this field", Line: 2, Column: 1},
		},

		// name
		"name_missing.yml": ValidationResults{ValidationError{File: "", Key: "name", Description: "name is a required field", Line: 1, Column: 1}},
		"name_nil.yml":     ValidationResults{ValidationError{File: "", Key: "name", Description: "name is a required field", Line: 4, Column: 1}},
		"name_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "name", Description: "wrong type for this field", Line: 4, Column: 1},
			ValidationError{File: "", Key: "name", Description: "name is a required field", Line: 4, Column: 1},
		},

		// applicationSuite
		"applicationSuite_wrong_type.yml": ValidationResults{ValidationError{File: "", Key: "applicationSuite", Description: "wrong type for this field", Line: 4, Column: 1}},

		// url
		"url_missing.yml": ValidationResults{ValidationError{File: "", Key: "url", Description: "url is a required field", Line: 1, Column: 1}},
		"url_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "url", Description: "wrong type for this field", Line: 6, Column: 1},
			ValidationError{File: "", Key: "url", Description: "url is a required field", Line: 6, Column: 1},
		},
		"url_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "url", Description: "url must be a valid URL", Line: 6, Column: 1},
			ValidationError{File: "", Key: "url", Description: "'foobar' not reachable: missing URL scheme", Line: 6, Column: 1},
			ValidationError{File: "", Key: "url", Description: "is not a valid code repository", Line: 6, Column: 1},
		},

		// landingURL
		"landingURL_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "landingURL", Description: "wrong type for this field", Line: 8, Column: 1},
			ValidationWarning{File: "", Key: "usedBy", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 21, Column: 1},
			ValidationWarning{File: "", Key: "dependsOn.open", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 98, Column: 3},
			ValidationWarning{File: "", Key: "IT", Description: "This key is DEPRECATED and will be removed in the future. Migrate its keys as their warnings say, then drop it", Line: 114, Column: 1},
		},
		"landingURL_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "landingURL", Description: "landingURL must be an HTTP URL", Line: 8, Column: 1},
			ValidationError{File: "", Key: "landingURL", Description: "'???' not reachable: missing URL scheme", Line: 8, Column: 1},
			ValidationWarning{File: "", Key: "usedBy", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 21, Column: 1},
			ValidationWarning{File: "", Key: "description.en.videos", Description: "This key is DEPRECATED and will be removed in the future. Link videos from the documentation or the website instead", Line: 62, Column: 5},
			ValidationWarning{File: "", Key: "dependsOn.open", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 102, Column: 3},
			ValidationWarning{File: "", Key: "IT", Description: "This key is DEPRECATED and will be removed in the future. Migrate its keys as their warnings say, then drop it", Line: 118, Column: 1},
		},

		// isBasedOn
		"isBasedOn_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "isBasedOn.foobar", Description: "wrong type for this field", Line: 10, Column: 1},
		},
		"isBasedOn_bad_url_array.yml": ValidationResults{
			ValidationError{File: "", Key: "isBasedOn[1]", Description: "isBasedOn[1] must be a valid URL", Line: 11, Column: 5},
		},
		"isBasedOn_bad_url_string.yml": ValidationResults{
			ValidationWarning{File: "", Key: "isBasedOn", Description: "The single string form is DEPRECATED and will be removed in the future. Use a one element array instead", Line: 9, Column: 1},
			ValidationError{File: "", Key: "isBasedOn[0]", Description: "isBasedOn[0] must be a valid URL", Line: 9, Column: 1},
		},

		// softwareVersion
		"softwareVersion_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "softwareVersion", Description: "wrong type for this field", Line: 8, Column: 1},
		},

		// releaseDate
		"releaseDate_empty.yml": ValidationResults{ValidationError{File: "", Key: "releaseDate", Description: "releaseDate must be a date with format 'YYYY-MM-DD'", Line: 8, Column: 1}},
		"releaseDate_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "releaseDate", Description: "wrong type for this field", Line: 8, Column: 1},
		},
		"releaseDate_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "releaseDate", Description: "releaseDate must be a date with format 'YYYY-MM-DD'", Line: 8, Column: 1},
		},
		"releaseDate_datetime.yml": ValidationResults{
			ValidationError{File: "", Key: "releaseDate", Description: "releaseDate must be a date with format 'YYYY-MM-DD'", Line: 8, Column: 1},
		},

		// logo
		"logo_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "logo", Description: "wrong type for this field", Line: 18, Column: 1},
		},
		"logo_unsupported_extension.yml": ValidationResults{
			ValidationError{File: "", Key: "logo", Description: "invalid file extension for: " + cwd + "/testdata/v0/invalid/logo.mpg", Line: 18, Column: 1},
		},
		"logo_missing_file.yml": ValidationResults{
			ValidationError{File: "", Key: "logo", Description: "no such file: " + cwd + "/testdata/v0/invalid/no_such_file.png", Line: 18, Column: 1},
		},
		"logo_absolute_path.yml": ValidationResults{
			ValidationError{File: "", Key: "logo", Description: "is an absolute path. Only relative paths or HTTP(s) URLs allowed", Line: 18, Column: 1},
		},
		"logo_outside_repository.yml": ValidationResults{
			ValidationError{File: "", Key: "logo", Description: "refers to a file outside the repository", Line: 18, Column: 1},
		},
		"logo_file_scheme.yml": ValidationResults{
			ValidationError{File: "", Key: "logo", Description: "is a file:// URL. Only relative paths or HTTP(s) URLs allowed", Line: 18, Column: 1},
		},
		"logo_file_scheme2.yml": ValidationResults{
			ValidationError{File: "", Key: "logo", Description: "is a file:// URL. Only relative paths or HTTP(s) URLs allowed", Line: 18, Column: 1},
		},
		"logo_file_scheme3.yml": ValidationResults{
			ValidationError{File: "", Key: "logo", Description: "is a file:// URL. Only relative paths or HTTP(s) URLs allowed", Line: 18, Column: 1},
		},
		"logo_file_scheme_uppercase.yml": ValidationResults{
			ValidationError{File: "", Key: "logo", Description: "is a file:// URL. Only relative paths or HTTP(s) URLs allowed", Line: 18, Column: 1},
		},
		"logo_file_scheme_mixed_case.yml": ValidationResults{
			ValidationError{File: "", Key: "logo", Description: "is a file:// URL. Only relative paths or HTTP(s) URLs allowed", Line: 18, Column: 1},
		},
		"logo_ftp_scheme.yml": ValidationResults{
			ValidationError{File: "", Key: "logo", Description: "has an unsupported URL scheme 'ftp'. Only relative paths or HTTP(s) URLs allowed", Line: 18, Column: 1},
		},
		"logo_data_scheme.yml": ValidationResults{
			ValidationError{File: "", Key: "logo", Description: "has an unsupported URL scheme 'data'. Only relative paths or HTTP(s) URLs allowed", Line: 18, Column: 1},
		},
		// Local publiccode.yml and URL in logo: should look for the logo remotely
		"logo_missing_url.yml": ValidationResults{
			ValidationError{File: "", Key: "logo", Description: "downloading https://google.com/no_such_file.png: not found", Line: 18, Column: 1},
		},

		// monochromeLogo
		"monochromeLogo_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "monochromeLogo", Description: "wrong type for this field", Line: 18, Column: 1},
		},
		"monochromeLogo_unsupported_extension.yml": ValidationResults{
			ValidationWarning{File: "", Key: "monochromeLogo", Description: "This key is DEPRECATED and will be removed in the future. Use 'logo' instead", Line: 18, Column: 1},
			ValidationError{File: "", Key: "monochromeLogo", Description: "invalid file extension for: " + cwd + "/testdata/v0/invalid/monochromeLogo.mpg", Line: 18, Column: 1},
		},
		"monochromeLogo_missing_file.yml": ValidationResults{
			ValidationWarning{File: "", Key: "monochromeLogo", Description: "This key is DEPRECATED and will be removed in the future. Use 'logo' instead", Line: 18, Column: 1},
			ValidationError{File: "", Key: "monochromeLogo", Description: "no such file: " + cwd + "/testdata/v0/invalid/no_such_file.png", Line: 18, Column: 1},
		},
		"monochromeLogo_outside_repository.yml": ValidationResults{
			ValidationWarning{File: "", Key: "monochromeLogo", Description: "This key is DEPRECATED and will be removed in the future. Use 'logo' instead", Line: 18, Column: 1},
			ValidationError{File: "", Key: "monochromeLogo", Description: "refers to a file outside the repository", Line: 18, Column: 1},
		},

		// organisation
		"organisation_wrong_uri.yml": ValidationResults{
			ValidationError{File: "", Key: "organisation.uri", Description: "uri is not a valid URI", Line: 19, Column: 3},
		},
		"organisation_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "organisation[0]", Description: "wrong type for this field", Line: 18, Column: 1},
		},
		"organisation_uri_missing.yml": ValidationResults{
			ValidationError{File: "", Key: "organisation.uri", Description: "uri is a required field", Line: 18, Column: 3},
		},
		"organisation_uri_wrong_italian_pa.yml": ValidationResults{
			ValidationError{File: "", Key: "organisation.uri", Description: "uri is not a valid URI", Line: 20, Column: 3},
		},
		"organisation_uri_wrong_italian_pa2.yml": ValidationResults{
			ValidationError{File: "", Key: "organisation.uri", Description: "uri must be a valid Italian Public Administration Code (iPA) with format 'urn:x-italian-pa:[codiceIPA]' (see https://github.com/publiccodeyml/italian-organizations-ipa-vocabulary)", Line: 19, Column: 3},
		},

		// inputTypes
		"inputTypes_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "inputTypes[1]", Description: "inputTypes[1] is not a valid MIME type", Line: 16, Column: 7},
			ValidationWarning{File: "", Key: "inputTypes", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 14, Column: 1},
		},
		"inputTypes_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "inputTypes.foobar", Description: "wrong type for this field", Line: 15, Column: 1},
		},

		// outputTypes
		"outputTypes_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "outputTypes[1]", Description: "outputTypes[1] is not a valid MIME type", Line: 16, Column: 7},
			ValidationWarning{File: "", Key: "outputTypes", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 14, Column: 1},
		},
		"outputTypes_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "outputTypes.foobar", Description: "wrong type for this field", Line: 15, Column: 1},
		},

		// platforms
		"platforms_missing.yml": ValidationResults{ValidationError{File: "", Key: "platforms", Description: "platforms must contain more than 0 items", Line: 1, Column: 1}},
		"platforms_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "platforms", Description: "wrong type for this field", Line: 9, Column: 1},
			ValidationError{File: "", Key: "platforms", Description: "platforms must contain more than 0 items", Line: 9, Column: 1},
		},

		// categories
		"categories_invalid.yml": ValidationResults{ValidationError{File: "", Key: "categories[0]", Description: "categories[0] must be a valid category (see https://github.com/publiccodeyml/publiccode.yml/blob/main/docs/standard/categories-list.rst)", Line: 13, Column: 5}},

		// usedBy
		"usedBy_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "usedBy", Description: "wrong type for this field", Line: 14, Column: 1},
		},

		// fundedBy
		"fundedBy_wrong_uri.yml": ValidationResults{
			ValidationError{File: "", Key: "fundedBy[0].uri", Description: "uri is not a valid URI", Line: 19, Column: 5},
		},
		"fundedBy_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "fundedBy.name", Description: "wrong type for this field", Line: 18, Column: 1},
		},
		"fundedBy_uri_missing.yml": ValidationResults{
			ValidationError{File: "", Key: "fundedBy[0].uri", Description: "uri is a required field", Line: 18, Column: 5},
		},
		"fundedBy_uri_wrong_italian_pa.yml": ValidationResults{
			ValidationError{File: "", Key: "fundedBy[0].uri", Description: "uri is not a valid URI", Line: 20, Column: 5},
		},
		"fundedBy_uri_wrong_italian_pa2.yml": ValidationResults{
			ValidationError{File: "", Key: "fundedBy[0].uri", Description: "uri must be a valid Italian Public Administration Code (iPA) with format 'urn:x-italian-pa:[codiceIPA]' (see https://github.com/publiccodeyml/italian-organizations-ipa-vocabulary)", Line: 19, Column: 5},
		},

		// roadmap
		"roadmap_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "roadmap", Description: "roadmap must be an HTTP URL", Line: 4, Column: 1},
			ValidationError{File: "", Key: "roadmap", Description: "'foobar' not reachable: missing URL scheme", Line: 4, Column: 1},
		},
		"roadmap_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "roadmap", Description: "wrong type for this field", Line: 4, Column: 1},
		},

		// developmentStatus
		"developmentStatus_missing.yml": ValidationResults{
			ValidationError{File: "", Key: "developmentStatus", Description: "developmentStatus is a required field", Line: 1, Column: 1},
		},
		"developmentStatus_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "developmentStatus", Description: "developmentStatus must be one of the following: \"concept\", \"development\", \"beta\", \"stable\" or \"obsolete\"", Line: 16, Column: 1},
		},
		"developmentStatus_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "developmentStatus", Description: "wrong type for this field", Line: 16, Column: 1},
			ValidationError{File: "", Key: "developmentStatus", Description: "developmentStatus is a required field", Line: 16, Column: 1},
		},

		// softwareType
		"softwareType_missing.yml": ValidationResults{
			ValidationError{File: "", Key: "softwareType", Description: "softwareType is a required field", Line: 1, Column: 1},
		},
		"softwareType_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "softwareType", Description: "softwareType must be one of the following: \"standalone/mobile\", \"standalone/iot\", \"standalone/desktop\", \"standalone/web\", \"standalone/backend\", \"standalone/other\", \"addon\", \"library\" or \"configurationFiles\"", Line: 17, Column: 1},
		},
		"softwareType_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "softwareType", Description: "wrong type for this field", Line: 17, Column: 1},
			ValidationError{File: "", Key: "softwareType", Description: "softwareType is a required field", Line: 17, Column: 1},
		},

		// intendedAudience
		// intendedAudience.*
		"intendedAudience_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "intendedAudience", Description: "wrong type for this field", Line: 18, Column: 1},
		},
		"intendedAudience_countries_invalid_country.yml": ValidationResults{
			ValidationError{File: "", Key: "intendedAudience.countries[2]", Description: "countries[2] must be a valid ISO 3166-1 alpha-2 two-letter country code", Line: 22, Column: 7},
		},
		"intendedAudience_countries_invalid_iso_3166_1_alpha_2.yml": ValidationResults{
			ValidationError{File: "", Key: "intendedAudience.countries[2]", Description: "countries[2] must be a valid ISO 3166-1 alpha-2 two-letter country code", Line: 22, Column: 7},
		},
		"intendedAudience_countries_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "intendedAudience.countries", Description: "wrong type for this field", Line: 19, Column: 1},
		},
		"intendedAudience_unsupportedCountries_invalid_country.yml": ValidationResults{
			ValidationError{File: "", Key: "intendedAudience.unsupportedCountries[0]", Description: "unsupportedCountries[0] must be a valid ISO 3166-1 alpha-2 two-letter country code", Line: 20, Column: 7},
			ValidationWarning{File: "", Key: "intendedAudience.unsupportedCountries", Description: "This key is DEPRECATED and will be removed in the future. Use 'intendedAudience.countries' to list the targeted countries instead", Line: 18, Column: 3},
		},
		"intendedAudience_unsupportedCountries_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "intendedAudience.unsupportedCountries", Description: "wrong type for this field", Line: 19, Column: 1},
		},
		"intendedAudience_scope_invalid_scope.yml": ValidationResults{
			ValidationError{File: "", Key: "intendedAudience.scope[0]", Description: "scope[0] must be a valid scope (see https://github.com/publiccodeyml/publiccode.yml/blob/main/docs/standard/scope-list.rst)", Line: 20, Column: 9},
		},
		"intendedAudience_scope_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "intendedAudience.scope", Description: "wrong type for this field", Line: 19, Column: 1},
		},

		// description
		// description.*
		"description_invalid_language.yml": ValidationResults{
			ValidationError{File: "", Key: "description", Description: "description must be a valid BCP 47 language", Line: 18, Column: 1},
		},
		"description_en_features_empty.yml": ValidationResults{
			ValidationError{File: "", Key: "description.en.features", Description: "features must contain more than 0 items", Line: 39, Column: 5},
		},
		"description_en_localisedName_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "description.en.localisedName", Description: "wrong type for this field", Line: 21, Column: 1},
			ValidationError{File: "", Key: "description", Description: "description must contain more than 0 items", Line: 18, Column: 1},
		},
		"description_en_genericName_too_long.yml": ValidationResults{
			ValidationError{File: "", Key: "description.en.genericName", Description: "genericName must be a maximum of 35 characters in length", Line: 22, Column: 5},
			ValidationWarning{File: "", Key: "description.en.genericName", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 22, Column: 5},
		},
		"description_en_shortDescription_missing.yml": ValidationResults{
			ValidationError{File: "", Key: "description.en.shortDescription", Description: "shortDescription is a required field", Line: 20, Column: 5},
		},
		"description_en_longDescription_missing.yml": ValidationResults{
			ValidationError{File: "", Key: "description.en.longDescription", Description: "longDescription is a required field", Line: 20, Column: 5},
		},
		"description_en_longDescription_too_long.yml": ValidationResults{
			ValidationError{File: "", Key: "description.en.longDescription", Description: "longDescription must be a maximum of 10000 characters in length", Line: 27, Column: 5},
		},
		"description_en_longDescription_too_short.yml": ValidationResults{
			ValidationError{File: "", Key: "description.en.longDescription", Description: "longDescription must be at least 150 characters in length", Line: 27, Column: 5},
		},
		"description_en_longDescription_too_short_grapheme_clusters.yml": ValidationResults{
			ValidationError{File: "", Key: "description.en.longDescription", Description: "longDescription must be at least 150 characters in length", Line: 28, Column: 5},
		},
		"description_en_documentation_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "description.en.documentation", Description: "documentation must be an HTTP URL", Line: 25, Column: 5},
			ValidationError{File: "", Key: "description.en.documentation", Description: "'not_a_url' not reachable: missing URL scheme", Line: 25, Column: 5},
		},
		"description_en_documentation_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "description.en.documentation", Description: "wrong type for this field", Line: 25, Column: 1},
			ValidationError{File: "", Key: "description", Description: "description must contain more than 0 items", Line: 20, Column: 1},
		},
		"description_en_apiDocumentation_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "description.en.apiDocumentation", Description: "apiDocumentation must be an HTTP URL", Line: 41, Column: 5},
			ValidationError{File: "", Key: "description.en.apiDocumentation", Description: "'abc' not reachable: missing URL scheme", Line: 41, Column: 5},
		},
		"description_en_apiDocumentation_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "description.en.apiDocumentation", Description: "wrong type for this field", Line: 43, Column: 1},
			ValidationError{File: "", Key: "description", Description: "description must contain more than 0 items", Line: 20, Column: 1},
		},
		"description_en_screenshots_missing_file.yml": ValidationResults{
			ValidationError{File: "", Key: "description.en.screenshots[0]", Description: "'no_such_file.png' is not an image: no such file: " + cwd + "/testdata/v0/invalid/no_such_file.png", Line: 42, Column: 9},
		},
		"description_en_screenshots_outside_repository.yml": ValidationResults{
			ValidationError{File: "", Key: "description.en.screenshots[0]", Description: "refers to a file outside the repository", Line: 42, Column: 9},
		},
		"description_en_awards_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "description.en.awards", Description: "wrong type for this field", Line: 40, Column: 1},
			ValidationError{File: "", Key: "description", Description: "description must contain more than 0 items", Line: 18, Column: 1},
		},
		"description_en_videos_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "description.en.videos[0]", Description: "videos[0] must be an HTTP URL", Line: 41, Column: 9},
			ValidationError{File: "", Key: "description.en.videos[1]", Description: "videos[1] is a required field", Line: 43, Column: 8},
			ValidationError{File: "", Key: "description.en.videos[2]", Description: "videos[2] must be an HTTP URL", Line: 45, Column: 9},
			ValidationWarning{File: "", Key: "description.en.videos", Description: "This key is DEPRECATED and will be removed in the future. Link videos from the documentation or the website instead", Line: 39, Column: 5},
			ValidationError{File: "", Key: "description.en.videos[0]", Description: "'ABC' is not a valid video URL supporting oEmbed: invalid oEmbed link: ABC", Line: 41, Column: 9},
			ValidationError{File: "", Key: "description.en.videos[2]", Description: "'' is not a valid video URL supporting oEmbed: invalid oEmbed link: ", Line: 45, Column: 9},
		},
		"description_en_videos_invalid_oembed.yml": ValidationResults{
			ValidationWarning{File: "", Key: "description.en.videos", Description: "This key is DEPRECATED and will be removed in the future. Link videos from the documentation or the website instead", Line: 39, Column: 5},
			ValidationError{File: "", Key: "description.en.videos[0]", Description: "'https://google.com' is not a valid video URL supporting oEmbed: invalid oEmbed link: https://google.com", Line: 41, Column: 9},
		},

		// legal
		// legal.*
		"legal_missing.yml": ValidationResults{ValidationError{File: "", Key: "legal.license", Description: "license is a required field", Line: 0, Column: 0}},
		"legal_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "legal", Description: "wrong type for this field", Line: 46, Column: 1},
			ValidationError{File: "", Key: "legal.license", Description: "license is a required field", Line: 46, Column: 8},
		},
		"legal_license_missing.yml": ValidationResults{ValidationError{File: "", Key: "legal.license", Description: "license is a required field", Line: 41, Column: 3}},
		"legal_license_invalid.yml": ValidationResults{ValidationError{File: "", Key: "legal.license", Description: "license must be a valid license (see https://spdx.org/licenses)", Line: 42, Column: 3}},
		"legal_authorsFile_missing_file.yml": ValidationResults{
			ValidationWarning{File: "", Key: "legal.authorsFile", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 42, Column: 3},
			ValidationError{File: "", Key: "legal.authorsFile", Description: "'" + cwd + "/testdata/v0/invalid/no_such_authors_file.txt' does not exist: no such file: " + cwd + "/testdata/v0/invalid/no_such_authors_file.txt", Line: 42, Column: 3},
		},
		"legal_authorsFile_file_scheme_uppercase.yml": ValidationResults{
			ValidationWarning{File: "", Key: "legal.authorsFile", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 42, Column: 3},
			ValidationError{File: "", Key: "legal.authorsFile", Description: "is a file:// URL. Only relative paths or HTTP(s) URLs allowed", Line: 42, Column: 3},
		},

		// maintenance
		// maintenance.*
		"maintenance_type_missing.yml": ValidationResults{
			ValidationError{File: "", Key: "maintenance.type", Description: "type is a required field", Line: 47, Column: 3},
		},
		"maintenance_type_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "maintenance.type", Description: "type must be one of the following: \"internal\", \"contract\", \"community\" or \"none\"", Line: 45, Column: 3},
		},
		"maintenance_contacts_missing_with_type_community.yml": ValidationResults{
			ValidationError{File: "", Key: "maintenance.contacts", Description: "contacts is a required field when \"type\" is \"community\"", Line: 44, Column: 3},
		},
		"maintenance_contacts_missing_with_type_internal.yml": ValidationResults{
			ValidationError{File: "", Key: "maintenance.contacts", Description: "contacts is a required field when \"type\" is \"internal\"", Line: 44, Column: 3},
		},
		"maintenance_contacts_name_missing.yml": ValidationResults{
			ValidationError{File: "", Key: "maintenance.contacts[0].name", Description: "name is a required field", Line: 47, Column: 7},
		},
		"maintenance_contacts_email_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "maintenance.contacts[0].email", Description: "email must be a valid email address", Line: 49, Column: 9},
		},
		"maintenance_contractors_missing_with_type_contract.yml": ValidationResults{
			ValidationError{File: "", Key: "maintenance.contractors", Description: "contractors is a required field when \"type\" is \"contract\"", Line: 44, Column: 3},
		},
		"maintenance_contractors_invalid_type.yml": ValidationResults{
			ValidationError{File: "", Key: "maintenance.contractors", Description: "wrong type for this field", Line: 47, Column: 1},
			ValidationError{File: "", Key: "maintenance.type", Description: "type is a required field", Line: 44, Column: 3},
		},
		"maintenance_contractors_name_missing.yml": ValidationResults{
			ValidationError{File: "", Key: "maintenance.contractors[0].name", Description: "name is a required field", Line: 47, Column: 7},
		},
		"maintenance_contractors_until_missing.yml": ValidationResults{
			ValidationError{File: "", Key: "maintenance.contractors[0].until", Description: "until is a required field", Line: 47, Column: 7},
		},
		"maintenance_contractors_until_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "maintenance.contractors[0].until", Description: "until must be a date with format 'YYYY-MM-DD'", Line: 49, Column: 7},
		},
		"maintenance_contractors_email_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "maintenance.contractors[0].email", Description: "email must be a valid email address", Line: 50, Column: 8},
		},
		"maintenance_contractors_website_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "maintenance.contractors[0].website", Description: "website must be an HTTP URL", Line: 52, Column: 7},
		},
		"maintenance_contractors_when_type_is_community.yml": ValidationResults{
			ValidationError{File: "", Key: "maintenance.contractors", Description: "contractors must not be present unless \"type\" is \"contract\"", Line: 46, Column: 3},
		},
		"maintenance_contractors_when_type_is_internal.yml": ValidationResults{
			ValidationError{File: "", Key: "maintenance.contractors", Description: "contractors must not be present unless \"type\" is \"contract\"", Line: 46, Column: 3},
		},
		"maintenance_contractors_when_type_is_none.yml": ValidationResults{
			ValidationError{File: "", Key: "maintenance.contractors", Description: "contractors must not be present unless \"type\" is \"contract\"", Line: 46, Column: 3},
		},

		// localisation
		"localisation_availableLanguages_missing.yml": ValidationResults{
			ValidationError{File: "", Key: "localisation.availableLanguages", Description: "availableLanguages is a required field", Line: 50, Column: 3},
		},
		"localisation_availableLanguages_empty.yml": ValidationResults{
			ValidationError{File: "", Key: "localisation.availableLanguages", Description: "availableLanguages must contain more than 0 items", Line: 52, Column: 3},
		},
		"localisation_availableLanguages_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "localisation.availableLanguages[0]", Description: "availableLanguages[0] must be a valid BCP 47 language", Line: 53, Column: 8},
		},
		"localisation_localisationReady_missing.yml": ValidationResults{
			ValidationError{File: "", Key: "localisation.localisationReady", Description: "localisationReady is a required field", Line: 52, Column: 3},
		},

		// dependsOn
		"dependsOn_open_name_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "dependsOn.open[0]", Description: "wrong type for this field", Line: 56, Column: 1},
		},
		"dependsOn_open_versionMin_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "dependsOn.open[0].versionMin", Description: "wrong type for this field", Line: 57, Column: 1},
		},
		"dependsOn_open_versionMax_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "dependsOn.open[0].versionMax", Description: "wrong type for this field", Line: 57, Column: 1},
		},
		"dependsOn_open_version_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "dependsOn.open[0].version", Description: "wrong type for this field", Line: 57, Column: 1},
		},
		"dependsOn_open_optional_wrong_type.yml": ValidationResults{
			ValidationError{File: "", Key: "dependsOn.open[0].optional", Description: "wrong type for this field", Line: 57, Column: 1},
		},

		// it.*
		"it_countryExtensionVersion_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "IT.countryExtensionVersion", Description: "countryExtensionVersion must be one of the following: \"0.2\" or \"1.0\"", Line: 12, Column: 3},
			ValidationWarning{File: "", Key: "IT", Description: "This key is DEPRECATED and will be removed in the future. Migrate its keys as their warnings say, then drop it", Line: 11, Column: 1},
		},
		"it_riuso_codiceIPA_invalid.yml": ValidationResults{
			ValidationError{File: "", Key: "IT.riuso.codiceIPA", Description: "codiceIPA must be a valid Italian Public Administration Code (iPA) (see https://github.com/publiccodeyml/italian-organizations-ipa-vocabulary)", Line: 55, Column: 5},
			ValidationWarning{File: "", Key: "IT", Description: "This key is DEPRECATED and will be removed in the future. Migrate its keys as their warnings say, then drop it", Line: 53, Column: 1},
		},
		"it_IT_duplicated.yml": ValidationResults{
			ValidationWarning{File: "", Key: "usedBy", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 19, Column: 1},
			ValidationWarning{File: "", Key: "description.en.videos", Description: "This key is DEPRECATED and will be removed in the future. Link videos from the documentation or the website instead", Line: 60, Column: 5},
			ValidationWarning{File: "", Key: "dependsOn.open", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 85, Column: 3},
			ValidationWarning{File: "", Key: "IT", Description: "This key is DEPRECATED and will be removed in the future. Migrate its keys as their warnings say, then drop it", Line: 105, Column: 1},
			ValidationWarning{File: "", Key: "it", Description: "This key is DEPRECATED and will be removed in the future. Migrate its keys as their warnings say, then drop it", Line: 114, Column: 1},
			ValidationError{File: "", Key: "it", Description: "'IT' key already present. Remove this key", Line: 114, Column: 1},
		},
		"it_wrong_case.yml": ValidationResults{
			ValidationError{File: "", Key: "It", Description: "unknown field \"It\"", Line: 105, Column: 1},
			ValidationWarning{File: "", Key: "usedBy", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 19, Column: 1},
			ValidationWarning{File: "", Key: "description.en.videos", Description: "This key is DEPRECATED and will be removed in the future. Link videos from the documentation or the website instead", Line: 60, Column: 5},
			ValidationWarning{File: "", Key: "dependsOn.open", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 85, Column: 3},
		},
		"description_en_gb_invalid_bcp47.yml": ValidationResults{
			ValidationError{File: "", Key: "description", Description: "description must be a valid BCP 47 language", Line: 18, Column: 1},
		},
		"localisation_availableLanguages_invalid_bcp47.yml": ValidationResults{
			ValidationError{File: "", Key: "localisation.availableLanguages[0]", Description: "availableLanguages[0] must be a valid BCP 47 language", Line: 54, Column: 8},
		},

		// misc
		"aliases_nested.yml": ValidationResults{
			ValidationError{File: "", Key: "zl7", Description: "YAML aliases expand to more than 10000 nodes", Line: 61, Column: 16},
		},
		"file_encoding.yml": ValidationResults{ValidationError{File: "", Key: "", Description: "Invalid UTF-8", Line: 0, Column: 0}},
		"invalid_yaml.yml":  ValidationResults{ValidationError{File: "", Key: "", Description: "value is not allowed in this context. map key-value is pre-defined", Line: 38, Column: 1}},
		"mostly_empty.yml": ValidationResults{
			ValidationError{File: "", Key: "name", Description: "name is a required field", Line: 1, Column: 1},
			ValidationError{File: "", Key: "url", Description: "url is a required field", Line: 1, Column: 1},
			ValidationError{File: "", Key: "platforms", Description: "platforms must contain more than 0 items", Line: 1, Column: 1},
			ValidationError{File: "", Key: "developmentStatus", Description: "developmentStatus is a required field", Line: 1, Column: 1},
			ValidationError{File: "", Key: "softwareType", Description: "softwareType is a required field", Line: 1, Column: 1},
			ValidationError{File: "", Key: "description[en-US].shortDescription", Description: "shortDescription is a required field", Line: 3, Column: 10},
			ValidationError{File: "", Key: "description[en-US].longDescription", Description: "longDescription is a required field", Line: 3, Column: 10},
			ValidationError{File: "", Key: "localisation.localisationReady", Description: "localisationReady is a required field", Line: 4, Column: 15},
			ValidationError{File: "", Key: "localisation.availableLanguages", Description: "availableLanguages is a required field", Line: 4, Column: 15},
			ValidationError{File: "", Key: "legal.license", Description: "license is a required field", Line: 5, Column: 8},
			ValidationError{File: "", Key: "maintenance.type", Description: "type is a required field", Line: 6, Column: 14},
		},
		"unknown_field.yml": ValidationResults{
			ValidationError{File: "", Key: "foobar", Description: "unknown field \"foobar\"", Line: 10, Column: 1},
		},
		"unknown_fields.yml": ValidationResults{
			ValidationError{File: "", Key: "zzzTop", Description: "unknown field \"zzzTop\"", Line: 8, Column: 1},
			ValidationError{File: "", Key: "description.en-GB.zzzDescription", Description: "unknown field \"zzzDescription\"", Line: 23, Column: 1},
			ValidationError{File: "", Key: "maintenance.contacts[0].zzzContact", Description: "unknown field \"zzzContact\"", Line: 51, Column: 1},
			ValidationError{File: "", Key: "zzzLast", Description: "unknown field \"zzzLast\"", Line: 58, Column: 1},
			ValidationError{File: "", Key: "description", Description: "description must contain more than 0 items", Line: 20, Column: 1},
			ValidationError{File: "", Key: "maintenance.type", Description: "type is a required field", Line: 47, Column: 3},
		},
	}

	dir := "testdata/v0/invalid/"
	testFiles, _ := filepath.Glob(dir + "*yml")
	for _, file := range testFiles {
		baseName := path.Base(file)
		if expected[baseName] == nil {
			t.Errorf("No expected data for file %s", baseName)
		}
		t.Run(file, func(t *testing.T) {
			err := parse(file)
			checkParseErrors(t, err, testType{file, expected[baseName]})
		})
	}
	for file := range expected {
		if !slices.Contains(testFiles, dir+file) {
			t.Errorf("No expected file %s", dir+file)
		}
	}
}

// Test v0 valid YAML testcases (testdata/v0/valid/).
func TestValidTestcasesV0(t *testing.T) {
	checkValidFiles(t, "testdata/v0/valid/*.yml")
}

// Test v0 valid YAML testcases (testdata/v0/valid_with_warnings/).
func TestValidWithWarningsTestcasesV0(t *testing.T) {
	expected := map[string]error{
		"unicode_grapheme_clusters.yml": ValidationResults{
			ValidationWarning{File: "", Key: "description.en.genericName", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 23, Column: 5},
		},
		"maintenance_contacts_phone_not_e164.yml": ValidationResults{
			ValidationWarning{File: "", Key: "maintenance.contacts[0].phone", Description: "Phone numbers SHOULD use the E.164 format, e.g. '+3923113215112'", Line: 46, Column: 7},
		},
		"valid.minimal.v0.2.yml": ValidationResults{
			ValidationWarning{File: "", Key: "publiccodeYmlVersion", Description: "v0.2 is not the latest version, use '0'. Parsing this file as v0.8.", Line: 1, Column: 1},
		},
		"valid.minimal.v0.3.yml": ValidationResults{
			ValidationWarning{File: "", Key: "publiccodeYmlVersion", Description: "v0.3 is not the latest version, use '0'. Parsing this file as v0.8.", Line: 1, Column: 1},
		},
		"valid.minimal.v0.4.yml": ValidationResults{
			ValidationWarning{File: "", Key: "publiccodeYmlVersion", Description: "v0.4 is not the latest version, use '0'. Parsing this file as v0.8.", Line: 1, Column: 1},
		},
		"valid.minimal.v0.7.yml": ValidationResults{
			ValidationWarning{File: "", Key: "publiccodeYmlVersion", Description: "v0.7 is not the latest version, use '0'. Parsing this file as v0.8.", Line: 1, Column: 1},
		},
		"valid.mime_types.yml": ValidationResults{
			ValidationWarning{File: "", Key: "inputTypes", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 48, Column: 1},
			ValidationWarning{File: "", Key: "outputTypes", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 50, Column: 1},
		},
		"valid_with_it_conforme_gdpr.yml": ValidationResults{
			ValidationWarning{File: "", Key: "IT", Description: "This key is DEPRECATED and will be removed in the future. Migrate its keys as their warnings say, then drop it", Line: 101, Column: 1},
			ValidationWarning{File: "", Key: "IT.conforme.gdpr", Description: "This key is DEPRECATED and will be removed in the future. Add a 'supports' entry with id 'alias:gdpr' instead", Line: 108, Column: 5},
		},
		"valid_with_it_conforme.yml": ValidationResults{
			ValidationWarning{File: "", Key: "IT", Description: "This key is DEPRECATED and will be removed in the future. Migrate its keys as their warnings say, then drop it", Line: 101, Column: 1},
			ValidationWarning{File: "", Key: "IT.conforme", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 103, Column: 3},
		},
		"valid_with_country_specific_section_downcase.yml": ValidationResults{
			ValidationWarning{File: "", Key: "it", Description: "This key is DEPRECATED and will be removed in the future. Migrate its keys as their warnings say, then drop it", Line: 91, Column: 1},
		},
		"valid_with_lowercase_countries.yml": ValidationResults{
			ValidationWarning{File: "", Key: "intendedAudience.countries[0]", Description: "Lowercase country codes are DEPRECATED. Use uppercase instead ('IT')", Line: 27, Column: 7},
			ValidationWarning{File: "", Key: "intendedAudience.countries[1]", Description: "Lowercase country codes are DEPRECATED. Use uppercase instead ('DE')", Line: 28, Column: 7},
			ValidationWarning{File: "", Key: "intendedAudience.unsupportedCountries", Description: "This key is DEPRECATED and will be removed in the future. Use 'intendedAudience.countries' to list the targeted countries instead", Line: 29, Column: 3},
			ValidationWarning{File: "", Key: "intendedAudience.unsupportedCountries[0]", Description: "Lowercase country codes are DEPRECATED. Use uppercase instead ('US')", Line: 30, Column: 7},
		},
		"valid_with_government_scope.yml": ValidationResults{
			ValidationWarning{File: "", Key: "intendedAudience.scope[1]", Description: "The 'government' scope is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 19, Column: 7},
		},
		"valid_with_unsupportedCountries.yml": ValidationResults{
			ValidationWarning{File: "", Key: "intendedAudience.unsupportedCountries", Description: "This key is DEPRECATED and will be removed in the future. Use 'intendedAudience.countries' to list the targeted countries instead", Line: 19, Column: 3},
		},
		"valid_with_isBasedOn_string.yml": ValidationResults{
			ValidationWarning{File: "", Key: "isBasedOn", Description: "The single string form is DEPRECATED and will be removed in the future. Use a one element array instead", Line: 6, Column: 1},
		},
		"valid_with_legal_repoOwner.yml": ValidationResults{
			ValidationWarning{File: "", Key: "legal.repoOwner", Description: "This key is DEPRECATED and will be removed in the future. Use 'organisation.name' instead", Line: 62, Column: 3},
		},
		"valid_with_usedBy.yml": ValidationResults{
			ValidationWarning{File: "", Key: "usedBy", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 12, Column: 1},
		},
		"valid_with_dependsOn_open.yml": ValidationResults{
			ValidationWarning{File: "", Key: "dependsOn.open", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 53, Column: 3},
		},
		"valid_with_videos.yml": ValidationResults{
			ValidationWarning{File: "", Key: "description.en-GB.videos", Description: "This key is DEPRECATED and will be removed in the future. Link videos from the documentation or the website instead", Line: 37, Column: 5},
		},
		"valid_with_IT_piattaforme.yml": ValidationResults{
			ValidationWarning{File: "", Key: "IT", Description: "This key is DEPRECATED and will be removed in the future. Migrate its keys as their warnings say, then drop it", Line: 215, Column: 1},
			ValidationWarning{File: "", Key: "IT.piattaforme.spid", Description: "This key is DEPRECATED and will be removed in the future. Add a 'supports' entry with id 'alias:spid' instead", Line: 217, Column: 5},
			ValidationWarning{File: "", Key: "IT.piattaforme.pagopa", Description: "This key is DEPRECATED and will be removed in the future. Add a 'supports' entry with id 'alias:pagopa' instead", Line: 218, Column: 5},
			ValidationWarning{File: "", Key: "IT.piattaforme.cie", Description: "This key is DEPRECATED and will be removed in the future. Add a 'supports' entry with id 'alias:cie' instead", Line: 219, Column: 5},
			ValidationWarning{File: "", Key: "IT.piattaforme.anpr", Description: "This key is DEPRECATED and will be removed in the future. Add a 'supports' entry with id 'alias:anpr' instead", Line: 220, Column: 5},
			ValidationWarning{File: "", Key: "IT.piattaforme.io", Description: "This key is DEPRECATED and will be removed in the future. Add a 'supports' entry with id 'alias:io' instead", Line: 221, Column: 5},
		},
		"valid_with_IT_riuso_codiceIPA.yml": ValidationResults{
			ValidationWarning{File: "", Key: "IT", Description: "This key is DEPRECATED and will be removed in the future. Migrate its keys as their warnings say, then drop it", Line: 101, Column: 1},
			ValidationWarning{File: "", Key: "IT.riuso.codiceIPA", Description: "This key is DEPRECATED and will be removed in the future. Use 'organisation.uri' and set it to 'urn:x-italian-pa:pcm' instead", Line: 103, Column: 5},
		},
		"valid_with_country_specific_section_uppercase.yml": ValidationResults{
			ValidationWarning{File: "", Key: "IT", Description: "This key is DEPRECATED and will be removed in the future. Migrate its keys as their warnings say, then drop it", Line: 90, Column: 1},
		},
		"countryExtensionVersion_1.0.yml": ValidationResults{
			ValidationWarning{File: "", Key: "IT", Description: "This key is DEPRECATED and will be removed in the future. Migrate its keys as their warnings say, then drop it", Line: 90, Column: 1},
		},
		"valid_no_it_conforme.yml": ValidationResults{
			ValidationWarning{File: "", Key: "IT", Description: "This key is DEPRECATED and will be removed in the future. Migrate its keys as their warnings say, then drop it", Line: 90, Column: 1},
		},
	}

	dir := "testdata/v0/valid_with_warnings/"
	testFiles, _ := filepath.Glob(dir + "*yml")
	for _, file := range testFiles {
		baseName := path.Base(file)
		if expected[baseName] == nil {
			t.Errorf("No expected data for file %s", baseName)
		}
		t.Run(file, func(t *testing.T) {
			err := parse(file)
			checkParseErrors(t, err, testType{file, expected[baseName]})
		})
	}
	for file := range expected {
		if !slices.Contains(testFiles, dir+file) {
			t.Errorf("No expected file %s", dir+file)
		}
	}
}

// Test publiccode.yml remote files for key errors.
func TestDecodeValueErrorsRemote(t *testing.T) {
	testRemoteFiles := []testType{
		// Pinned to a commit: fetching from main means a testdata change
		// breaks this test in every PR until the change itself lands.
		{"https://raw.githubusercontent.com/italia/publiccode-parser-go/389af926bcb5bcb723b56d8e390585f2d0f2c7e8/testdata/v0/valid_with_warnings/valid_with_lowercase_countries.yml", ValidationResults{
			ValidationWarning{File: "", Key: "isBasedOn", Description: "The single string form is DEPRECATED and will be removed in the future. Use a one element array instead", Line: 7, Column: 1},
			ValidationWarning{File: "", Key: "usedBy", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 18, Column: 1},
			ValidationWarning{File: "", Key: "intendedAudience.countries[0]", Description: "Lowercase country codes are DEPRECATED. Use uppercase instead ('IT')", Line: 31, Column: 7},
			ValidationWarning{File: "", Key: "intendedAudience.countries[1]", Description: "Lowercase country codes are DEPRECATED. Use uppercase instead ('DE')", Line: 32, Column: 7},
			ValidationWarning{File: "", Key: "intendedAudience.unsupportedCountries", Description: "This key is DEPRECATED and will be removed in the future. Use 'intendedAudience.countries' to list the targeted countries instead", Line: 33, Column: 3},
			ValidationWarning{File: "", Key: "intendedAudience.unsupportedCountries[0]", Description: "Lowercase country codes are DEPRECATED. Use uppercase instead ('US')", Line: 34, Column: 7},
			ValidationWarning{File: "", Key: "description.en.videos", Description: "This key is DEPRECATED and will be removed in the future. Link videos from the documentation or the website instead", Line: 62, Column: 5},
			ValidationWarning{File: "", Key: "maintenance.contacts[0].phone", Description: "Phone numbers SHOULD use the E.164 format, e.g. '+3923113215112'", Line: 83, Column: 7},
			ValidationWarning{File: "", Key: "maintenance.contacts[1].phone", Description: "Phone numbers SHOULD use the E.164 format, e.g. '+3923113215112'", Line: 87, Column: 7},
			ValidationWarning{File: "", Key: "maintenance.contacts[2].phone", Description: "Phone numbers SHOULD use the E.164 format, e.g. '+3923113215112'", Line: 91, Column: 7},
			ValidationWarning{File: "", Key: "dependsOn.open", Description: "This key is DEPRECATED and will be removed in the future. It's safe to drop it", Line: 102, Column: 3},
		}},
	}

	parser, err := NewDefaultParser()
	if err != nil {
		t.Errorf("Can't create parser: %v", err)
	}

	for _, test := range testRemoteFiles {
		t.Run(fmt.Sprintf("%v", test.err), func(t *testing.T) {
			var err error

			_, err = parser.Parse(test.file)

			checkParseErrors(t, err, test)
		})
	}
}

// Test that files in fields with relative paths or URLs (ie. logo, screenshots, etc.)
// are checked and expanded correctly.
func TestRelativePathsOrURLs(t *testing.T) {
	testRemoteFiles := []testType{
		// Remote publiccode.yml and relative path in screenshots:
		// should look for the screenshot remotely relative to this URL
		{"https://raw.githubusercontent.com/italia/publiccode-parser-go/2a1f25a36e0f2ab2d71efb67db8b66ff00573fbe/testdata/v0/invalid/description_en_screenshots_missing_file.yml", ValidationResults{
			ValidationError{File: "", Key: "description.en.screenshots[0]", Description: "'no_such_file.png' is not an image: downloading https://raw.githubusercontent.com/italia/publiccode-parser-go/2a1f25a36e0f2ab2d71efb67db8b66ff00573fbe/testdata/v0/invalid/no_such_file.png: not found", Line: 42, Column: 9},
		}},

		// Local publiccode.yml and relative path in screenshot:
		// should look for the logo relative to this path in the filesystem
		{"testdata/v0/invalid/description_en_screenshots_missing_file.yml", ValidationResults{
			ValidationError{File: "", Key: "description.en.screenshots[0]", Description: "'no_such_file.png' is not an image: no such file: " + cwd + "/testdata/v0/invalid/no_such_file.png", Line: 42, Column: 9},
		}},

		// Remote publiccode.yml and URL in logo:
		// should look for the logo remotely
		{"https://raw.githubusercontent.com/italia/publiccode-parser-go/2a1f25a36e0f2ab2d71efb67db8b66ff00573fbe/testdata/v0/invalid/logo_missing_url.yml", ValidationResults{
			ValidationError{File: "", Key: "logo", Description: "downloading https://google.com/no_such_file.png: not found", Line: 18, Column: 1},
		}},

		// Local publiccode.yml and URL in logo:
		// should look for the logo remotely
		//
		// (already tested in TestInvalidTestcasesV0)
		// "testdata/v0/invalid/logo_missing_url.yml", ValidationResults{
		//	ValidationError{"logo", "downloading https://google.com/no_such_file.png: not found", 18, 1},
		// }},
	}

	parser, err := NewDefaultParser()
	if err != nil {
		t.Errorf("Can't create parser: %v", err)
	}

	for _, test := range testRemoteFiles {
		t.Run(fmt.Sprintf("%v", test.err), func(t *testing.T) {
			var err error

			_, err = parser.Parse(test.file)

			checkParseErrors(t, err, test)
		})
	}
}

// Test that files in fields with relative paths or URLs (ie. logo, screenshots, etc.)
// are checked and expanded correctly when DisableNetwork is true.
func TestRelativePathsOrURLsNoNetworkRemoteChecks(t *testing.T) {
	testRemoteFiles := []string{
		// Remote publiccode.yml and relative path in screenshots:
		// should look for the screenshot remotely relative to this URL,
		// but DisableNetwork is true, so no check is performed.
		"https://raw.githubusercontent.com/italia/publiccode-parser-go/2a1f25a36e0f2ab2d71efb67db8b66ff00573fbe/testdata/v0/invalid/description_en_screenshots_missing_file.yml",

		// Remote publiccode.yml and URL in logo:
		// should look for the logo remotely but DisableNetwork is true, so no check is performed.
		"https://raw.githubusercontent.com/italia/publiccode-parser-go/2a1f25a36e0f2ab2d71efb67db8b66ff00573fbe/testdata/v0/invalid/logo_missing_url.yml",

		// Local publiccode.yml and URL in logo:
		// should look for the logo remotely but DisableNetwork is true, so no check is performed.
		"testdata/v0/invalid/logo_missing_url.yml",
	}

	for _, file := range testRemoteFiles {
		t.Run(file, func(t *testing.T) {
			if err := parseNoNetwork(file); err != nil {
				t.Errorf("[%s] validation failed for valid file: %T - %s\n", file, err, err)
			}
		})
	}
}

// Test that files in fields with relative paths or URLs (ie. logo, screenshots, etc.)
// are checked and expanded correctly when DisableNetwork is true.
func TestRelativePathsOrURLsNoNetwork(t *testing.T) {
	testFiles := []testType{
		// Local publiccode.yml and relative path in screenshot:
		// should look for the logo relative to this path in the filesystem.
		// DisableNetwork doesn't affect this so the check *IS* performed.
		{"testdata/v0/invalid/description_en_screenshots_missing_file.yml", ValidationResults{
			ValidationError{File: "", Key: "description.en.screenshots[0]", Description: "'no_such_file.png' is not an image: no such file: " + cwd + "/testdata/v0/invalid/no_such_file.png", Line: 42, Column: 9},
		}},
	}

	parser, err := NewParser(ParserConfig{DisableNetwork: true})
	if err != nil {
		t.Errorf("Can't create parser: %v", err)
	}

	for _, test := range testFiles {
		t.Run(fmt.Sprintf("%v", test.err), func(t *testing.T) {
			var err error

			_, err = parser.Parse(test.file)

			checkParseErrors(t, err, test)
		})
	}
}

func TestUrlMissingWithoutPath(t *testing.T) {
	expected := map[string]error{
		"url_missing.yml": ValidationResults{
			ValidationError{File: "", Key: "url", Description: "url is a required field", Line: 1, Column: 1},
		},
	}

	parser, err := NewDefaultParser()
	if err != nil {
		t.Errorf("Can't create parser: %v", err)
	}

	testFiles, _ := filepath.Glob("testdata/v0/invalid/url_missing.yml")
	for _, file := range testFiles {
		baseName := path.Base(file)
		if expected[baseName] == nil {
			t.Errorf("No expected data for file %s", baseName)
		}
		t.Run(file, func(t *testing.T) {
			_, err := parser.Parse(file)

			checkParseErrors(t, err, testType{file, expected[baseName]})
		})
	}
}
