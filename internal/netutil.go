package netutil

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/publiccodeyml/libpubliccode/v5/internal/httpclient"
)

// downloadFile download the file in the path.
func downloadFile(client *httpclient.Client, filepath string, url *url.URL, headers map[string]string) error {
	// Create the file.
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}

	defer func() {
		_ = out.Close()
	}()

	// Get the data from the url.
	body, err := client.Get(url.String(), headers)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", url, err)
	}

	// Write the body to file.
	if _, err = out.Write(body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// DownloadTmpFile downloads the passed URL to a temporary directory.
// Caller is responsible for removing the temporary directory.
func DownloadTmpFile(client *httpclient.Client, url *url.URL, headers map[string]string) (string, error) {
	// Create a temp dir
	tmpdir, err := os.MkdirTemp("", "publiccode.yml-parser-go")
	if err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}

	// Download the file in the temp dir.
	tmpFile := filepath.Join(tmpdir, path.Base(url.Path))

	err = downloadFile(client, tmpFile, url, headers)
	if err != nil {
		return "", err
	}

	return tmpFile, nil
}
