package module

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"golang.org/x/mod/module"
)

// Client represents a module client.
type Client interface {
	Version(path, ver string) (module.Version, error)
	Download(m module.Version) (io.ReadCloser, error)
}

// ProxyClient gets modules from a Go proxy.
type ProxyClient struct {
	url *url.URL
}

// NewProxyClient returns a proxy client for proxyURL.
func NewProxyClient(proxyURL string) (*ProxyClient, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy url: %w", err)
	}

	return &ProxyClient{
		url: u,
	}, nil
}

// Version resolves the version ver of path.
func (c *ProxyClient) Version(p, ver string) (module.Version, error) {
	var m module.Version
	p, err := module.EscapePath(p)
	if err != nil {
		return m, fmt.Errorf("invalid module path %q: %w", p, err)
	}
	m.Path = p
	v, err := module.EscapeVersion(ver)
	if err != nil {
		return m, fmt.Errorf("invalid module version %q: %w", ver, err)
	}

	rawpath := path.Join(p, "@v", v+".info")
	if ver == "latest" {
		rawpath = path.Join(p, "@latest")
	}
	u := *c.url
	u.Path = rawpath

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return m, fmt.Errorf("could not create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return m, fmt.Errorf("could not resolve version %q: %w", rawpath, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return m, fmt.Errorf("version %s does not exist for module %q", v, p)
	}
	if err = json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return m, fmt.Errorf("could not resolve version %q: %w", rawpath, err)
	}
	return m, nil
}

// Download returns a zip of the version and path m.
func (c *ProxyClient) Download(m module.Version) (io.ReadCloser, error) {
	rawpath := path.Join(m.Path, "@v", m.Version+".zip")
	u := *c.url
	u.Path = rawpath

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch module %q: %w", rawpath, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("version %s does not exist for module %q", m.Version, m.Path)
	}
	return resp.Body, nil
}

// CachedClient is a file caching client.
type CachedClient struct {
	path string
	c    Client
}

// NewCachedClient returns a caching client.
func NewCachedClient(c Client, cachePath string) (*CachedClient, error) {
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("cache path %q does not exist", cachePath)
	}

	return &CachedClient{
		path: cachePath,
		c:    c,
	}, nil
}

// Version resolves the version ver of path.
func (c *CachedClient) Version(path, ver string) (module.Version, error) {
	return c.c.Version(path, ver)
}

// Download returns a zip of the version and path m.
func (c *CachedClient) Download(m module.Version) (io.ReadCloser, error) {
	p := filepath.Join(c.path, m.Path, m.Version+".zip")
	if _, err := os.Stat(p); err == nil {
		if rc, err := os.Open(filepath.Clean(p)); err == nil {
			return rc, nil
		}
		// If there is an error opening the cache file we should replace it.
	}

	rc, err := c.c.Download(m)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rc.Close()
	}()

	if err = os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return nil, fmt.Errorf("could not create cache path %q", filepath.Dir(p))
	}
	f, err := os.OpenFile(filepath.Clean(p), os.O_CREATE|os.O_TRUNC|os.O_RDWR|os.O_EXCL, 0o444)
	if err != nil {
		return nil, fmt.Errorf("could not write to cache file %q: %w", p, err)
	}
	if _, err = io.Copy(f, rc); err != nil {
		return nil, fmt.Errorf("could not write to cache file %q: %w", p, err)
	}

	if _, err = f.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("could not read cache file %q: %w", p, err)
	}
	return f, err
}
