package publiccode_test

import (
	"errors"
	"fmt"
	"os"
	"strings"

	publiccode "github.com/publiccodeyml/libpubliccode/v5"
)

func ExampleNewDefaultParser() {
	parser, err := publiccode.NewDefaultParser()
	if err != nil {
		panic(err)
	}

	_, err = parser.Parse("testdata/v0/valid/valid.minimal.yml")
	if err != nil {
		fmt.Println(err)
	}
	// Output:
}

func ExampleNewParser_disableNetwork() {
	parser, err := publiccode.NewParser(publiccode.ParserConfig{
		DisableNetwork: true,
	})
	if err != nil {
		panic(err)
	}

	_, err = parser.Parse("testdata/v0/valid/valid.minimal.yml")
	if err != nil {
		fmt.Println(err)
	}
	// Output:
}

func ExampleParser_ParseStream() {
	yaml := `
publiccodeYmlVersion: "0"

name: My Software
url: "https://github.com/example/example"

platforms:
  - web

categories:
  - cloud-management

developmentStatus: development

softwareType: "standalone/other"

description:
  en-GB:
    localisedName: My Software
    shortDescription: >
          A rather short description which
          is probably useless
    longDescription: >
          Very long description of this software, also split
          on multiple rows. You should note what the software
          is and why one should need it. This is 158 characters.
          Very long description of this software, also split
          on multiple rows. You should note what the software
          is and why one should need it. This is 316 characters.
          Very long description of this software, also split
          on multiple rows. You should note what the software
          is and why one should need it. This is 474 characters.
          Very long description of this software, also split
          on multiple rows. You should note what the software
          is and why one should need it. This is 632 characters.
    features:
       - Just one feature

legal:
  license: AGPL-3.0-or-later

maintenance:
  type: "community"

  contacts:
    - name: Francesco Rossi

localisation:
  localisationReady: true
  availableLanguages:
    - en
`

	parser, err := publiccode.NewParser(publiccode.ParserConfig{
		DisableNetwork: true,
	})
	if err != nil {
		panic(err)
	}

	_, err = parser.ParseStream(strings.NewReader(yaml))
	if err != nil {
		fmt.Println(err)
	}
	// Output:
}

// A valid document can return warnings through the error result.
func ExampleParser_ParseStream_warnings() {
	data, err := os.ReadFile("testdata/v0/valid/valid.minimal.yml")
	if err != nil {
		panic(err)
	}

	// An older supported version produces a warning.
	input := strings.Replace(string(data), `publiccodeYmlVersion: "0"`, `publiccodeYmlVersion: "0.7"`, 1)
	parser, err := publiccode.NewParser(publiccode.ParserConfig{DisableNetwork: true})
	if err != nil {
		panic(err)
	}

	pc, err := parser.ParseStream(strings.NewReader(input))
	valid := pc != nil
	if err != nil {
		results, ok := errors.AsType[publiccode.ValidationResults](err)
		if !ok {
			fmt.Println(err)

			return
		}

		for _, result := range results {
			var diagnostic publiccode.ValidationWarning
			var validationError publiccode.ValidationError
			switch {
			case errors.As(result, &diagnostic):
				fmt.Println("warning:", diagnostic.Key)
			case errors.As(result, &validationError):
				fmt.Println("error:", validationError.Key)
				valid = false
			default:
				fmt.Println(result)
				valid = false
			}
		}
	}

	fmt.Println("valid:", valid)
	// Output:
	// warning: publiccodeYmlVersion
	// valid: true
}
