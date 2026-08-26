package netutil

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestBuildUserAgent(t *testing.T) {
	cases := []struct {
		name string
		info *debug.BuildInfo
		want string
	}{
		{
			name: "no build information",
			info: nil,
			want: "libpubliccode/devel (+https://github.com/publiccodeyml/libpubliccode)",
		},
		{
			name: "released CLI: this module is the main one",
			info: &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "v5.4.3"}},
			want: "libpubliccode/5.4.3 (+https://github.com/publiccodeyml/libpubliccode)",
		},
		{
			name: "unstamped build of the CLI",
			info: &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "(devel)"}},
			want: "libpubliccode/devel (+https://github.com/publiccodeyml/libpubliccode)",
		},
		{
			name: "embedded as a library: this module is a dependency",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "example.org/harvester", Version: "v1.0.0"},
				Deps: []*debug.Module{
					{Path: "example.org/other", Version: "v0.1.0"},
					{Path: modulePath, Version: "v5.4.3"},
				},
			},
			want: "libpubliccode/5.4.3 (+https://github.com/publiccodeyml/libpubliccode)",
		},
		{
			name: "pseudo-version identifies the commit and is kept",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "example.org/harvester", Version: "v1.0.0"},
				Deps: []*debug.Module{{Path: modulePath, Version: "v5.4.4-0.20260316100201-5dd490bc4896"}},
			},
			want: "libpubliccode/5.4.4-0.20260316100201-5dd490bc4896 " +
				"(+https://github.com/publiccodeyml/libpubliccode)",
		},
		{
			name: "CLI installed with \"go install\": the main module is the command",
			info: &debug.BuildInfo{Main: debug.Module{Path: modulePath + "/publiccode-parser", Version: "v5.4.3"}},
			want: "libpubliccode/5.4.3 (+https://github.com/publiccodeyml/libpubliccode)",
		},
		{
			name: "release build of the CLI: no module version in the build information",
			info: &debug.BuildInfo{Main: debug.Module{Path: "command-line-arguments"}},
			want: "libpubliccode/devel (+https://github.com/publiccodeyml/libpubliccode)",
		},
		{
			name: "this module is nowhere to be found",
			info: &debug.BuildInfo{Main: debug.Module{Path: "example.org/harvester", Version: "v1.0.0"}},
			want: "libpubliccode/devel (+https://github.com/publiccodeyml/libpubliccode)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildUserAgent(tc.info); got != tc.want {
				t.Errorf("buildUserAgent() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestUserAgentForVersionFormatsModuleVersions(t *testing.T) {
	cases := map[string]string{
		"5.4.3":   "libpubliccode/5.4.3 (+https://github.com/publiccodeyml/libpubliccode)",
		"v5.4.3":  "libpubliccode/5.4.3 (+https://github.com/publiccodeyml/libpubliccode)",
		"devel":   "libpubliccode/devel (+https://github.com/publiccodeyml/libpubliccode)",
		"(devel)": "libpubliccode/devel (+https://github.com/publiccodeyml/libpubliccode)",
		"":        "libpubliccode/devel (+https://github.com/publiccodeyml/libpubliccode)",
	}

	for version, want := range cases {
		if got := userAgentForVersion(version); got != want {
			t.Errorf("userAgentForVersion(%q) = %q, want %q", version, got, want)
		}
	}
}

func TestUserAgentNamesThisProject(t *testing.T) {
	got := UserAgent()

	if !strings.HasPrefix(got, productName+"/") {
		t.Errorf("UserAgent() = %q, want it to start with %q", got, productName+"/")
	}
	if !strings.Contains(got, projectURL) {
		t.Errorf("UserAgent() = %q, want it to point at %q", got, projectURL)
	}
}
