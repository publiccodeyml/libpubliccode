package publiccode

import (
	"errors"
	"testing"
)

func TestIsRelativePathOrURL(t *testing.T) {
	tests := []struct {
		content string
		err     string
	}{
		{"logo.png", ""},
		{"img/logo.png", ""},
		{"img/logo:1.png", ""},
		{"logo%.png", ""},
		{"https://example.org/logo.png", ""},
		{"http://example.org/logo.png", ""},
		{"HTTPS://example.org/logo.png", ""},
		{"/logo.png", "is an absolute path. Only relative paths or HTTP(s) URLs allowed"},
		{"file:///logo.png", "is a file:// URL. Only relative paths or HTTP(s) URLs allowed"},
		{"FILE:///logo.png", "is a file:// URL. Only relative paths or HTTP(s) URLs allowed"},
		{"File:///%zz.png", "is a file:// URL. Only relative paths or HTTP(s) URLs allowed"},
		{"ftp://example.org/logo.png", "has an unsupported URL scheme 'ftp'. Only relative paths or HTTP(s) URLs allowed"},
		{"data:image/png;base64,iVBORw0KGgo=", "has an unsupported URL scheme 'data'. Only relative paths or HTTP(s) URLs allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			ok, err := isRelativePathOrURL(tt.content, "logo")

			if tt.err == "" {
				if !ok || err != nil {
					t.Errorf("expected %q to be accepted, got %v", tt.content, err)
				}

				return
			}

			var ve ValidationError
			if ok || !errors.As(err, &ve) || ve.Key != "logo" || ve.Description != tt.err {
				t.Errorf("expected error %q for %q, got %v", tt.err, tt.content, err)
			}
		})
	}
}
