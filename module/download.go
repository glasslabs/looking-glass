package module

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// Downloader downloads and caches modules.
type Downloader struct {
	httpClient *http.Client
	cachePath  string
}

// NewDownloader returns a Downloader with the cachePath.
// If the cache path does not exist, the reader attempts to create it.
func NewDownloader(cachePath string) (*Downloader, error) {
	if err := ensurePath(cachePath); err != nil {
		return nil, err
	}

	cachePath, err := filepath.Abs(cachePath)
	if err != nil {
		return nil, err
	}

	return &Downloader{
		httpClient: http.DefaultClient,
		cachePath:  cachePath,
	}, nil
}

// Download fetches the file given in uri, caching it locally.
func (d *Downloader) Download(ctx context.Context, uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("parsing uri: %w", err)
	}

	switch u.Scheme {
	case "file", "":
		path := filepath.Join(d.cachePath, u.Path)
		if _, err = os.Stat(path); err != nil {
			return "", err
		}
		return u.Path, nil
	case "http", "https":
		return d.withCache(u.Path, func() ([]byte, error) {
			return d.readHTTPFile(ctx, uri)
		})
	default:
		return "", fmt.Errorf("unsupported uri scheme %q", u.Scheme)
	}
}

func (d *Downloader) withCache(path string, fn func() ([]byte, error)) (string, error) {
	cachedPath := filepath.Join(d.cachePath, path)

	if _, err := os.Stat(cachedPath); err == nil {
		return path, nil
	}
	b, err := fn()
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(cachedPath)
	if err = ensurePath(dir); err != nil {
		return "", err
	}

	if err = os.WriteFile(cachedPath, b, 0o644); err != nil {
		return "", fmt.Errorf("caching file %q: %w", path, err)
	}
	return path, nil
}

func (d *Downloader) readHTTPFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func ensurePath(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(path, 0o750); err != nil {
		return fmt.Errorf("creating path %q: %w", path, err)
	}
	return nil
}
