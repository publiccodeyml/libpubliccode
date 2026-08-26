package publiccode

import (
	urlutil "github.com/publiccodeyml/libpubliccode/v5/internal"
)

// UserAgent returns the value the parser sends in the User-Agent header of the
// requests it makes during the external checks, e.g.
//
//	libpubliccode/5.4.3 (+https://github.com/publiccodeyml/libpubliccode)
//
// The version comes from the build information of the running binary, and is
// "devel" when there is none to be found.
//
// Programs embedding the parser should identify themselves instead, through
// [ParserConfig.UserAgent]. This function is exported for the ones that want to
// keep this token and add their own to it.
func UserAgent() string {
	return urlutil.UserAgent()
}
