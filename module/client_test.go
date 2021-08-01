package module_test

import (
	"bytes"
	"errors"
	"io"

	"net/http"
	"os"
	"testing"

	"github.com/glasslabs/looking-glass/module"
	httptest "github.com/hamba/testutils/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mod "golang.org/x/mod/module"
)

func TestNewProxyClient(t *testing.T) {
	got, err := module.NewProxyClient("http://example.com")

	if assert.NoError(t, err) {
		assert.Implements(t, (*module.Client)(nil), got)
	}
}

func TestNewProxyClient_HandlesBadURL(t *testing.T) {
	_, err := module.NewProxyClient("ba\nlh")

	assert.Error(t, err)
}

func TestProxyClient_VersionResolvesVersion(t *testing.T) {
	srv := httptest.NewServer(t)
	srv.On(http.MethodGet, "/github.com/glasslabs/test/@v/main.info").
		ReturnsString(http.StatusOK, `{"Version":"v0.1.0"}`)
	t.Cleanup(func() {
		srv.Close()
	})

	c, err := module.NewProxyClient(srv.URL())
	require.NoError(t, err)

	got, err := c.Version("github.com/glasslabs/test", "main")

	if assert.NoError(t, err) {
		assert.Equal(t, "github.com/glasslabs/test", got.Path)
		assert.Equal(t, "v0.1.0", got.Version)
		srv.AssertExpectations()
	}
}

func TestProxyClient_VersionResolvesLatest(t *testing.T) {
	srv := httptest.NewServer(t)
	srv.On(http.MethodGet, "/github.com/glasslabs/test/@latest").
		ReturnsString(http.StatusOK, `{"Version":"v0.1.0"}`)
	t.Cleanup(func() {
		srv.Close()
	})

	c, err := module.NewProxyClient(srv.URL())
	require.NoError(t, err)

	got, err := c.Version("github.com/glasslabs/test", "latest")

	if assert.NoError(t, err) {
		assert.Equal(t, "github.com/glasslabs/test", got.Path)
		assert.Equal(t, "v0.1.0", got.Version)
		srv.AssertExpectations()
	}
}

func TestProxyClient_VersionHandlesError(t *testing.T) {
	srv := httptest.NewServer(t)
	srv.On(http.MethodGet, "/github.com/glasslabs/test/@v/main.info").
		ReturnsString(http.StatusNotFound, `Not Found`)
	t.Cleanup(func() {
		srv.Close()
	})

	c, err := module.NewProxyClient(srv.URL())
	require.NoError(t, err)

	_, err = c.Version("github.com/glasslabs/test", "main")

	assert.Error(t, err)
	srv.AssertExpectations()
}

func TestProxyClient_VersionHandlesBadJSON(t *testing.T) {
	srv := httptest.NewServer(t)
	srv.On(http.MethodGet, "/github.com/glasslabs/test/@v/main.info").
		ReturnsString(http.StatusOK, `{`)
	t.Cleanup(func() {
		srv.Close()
	})

	c, err := module.NewProxyClient(srv.URL())
	require.NoError(t, err)

	_, err = c.Version("github.com/glasslabs/test", "main")

	assert.Error(t, err)
	srv.AssertExpectations()
}

func TestProxyClient_Download(t *testing.T) {
	srv := httptest.NewServer(t)
	srv.On(http.MethodGet, "/test/@v/main.zip").
		ReturnsString(http.StatusOK, `1234`)
	t.Cleanup(func() {
		srv.Close()
	})

	c, err := module.NewProxyClient(srv.URL())
	require.NoError(t, err)

	got, err := c.Download(mod.Version{Path: "test", Version: "main"})

	if assert.NoError(t, err) {
		if !assert.Implements(t, (*io.ReadCloser)(nil), got) {
			return
		}
		data, _ := io.ReadAll(got)
		assert.Equal(t, "1234", string(data))
		srv.AssertExpectations()
	}
}

func TestProxyClient_DownloadHandlesError(t *testing.T) {
	srv := httptest.NewServer(t)
	srv.On(http.MethodGet, "/test/@v/main.zip").
		ReturnsStatus(http.StatusNotFound)
	t.Cleanup(func() {
		srv.Close()
	})

	c, err := module.NewProxyClient(srv.URL())
	require.NoError(t, err)

	_, err = c.Download(mod.Version{Path: "test", Version: "main"})

	assert.Error(t, err)
}

func TestNewCachedClient(t *testing.T) {
	dir, err := os.MkdirTemp("./", "cache-test")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	c := &MockClient{}

	got, err := module.NewCachedClient(c, dir)

	if assert.NoError(t, err) {
		assert.Implements(t, (*module.Client)(nil), got)
	}
}

func TestNewCachedClient_HandlesBadPath(t *testing.T) {
	c := &MockClient{}

	_, err := module.NewCachedClient(c, "something")

	assert.Error(t, err)
}

func TestCachedClient_Version(t *testing.T) {
	dir, err := os.MkdirTemp("./", "cache-test")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	c := &MockClient{}
	c.On("Version", "test", "main").Return(mod.Version{Path: "test", Version: "v0.1.0"}, nil)

	cache, err := module.NewCachedClient(c, dir)
	require.NoError(t, err)

	got, err := cache.Version("test", "main")

	if assert.NoError(t, err) {
		assert.Equal(t, "test", got.Path)
		assert.Equal(t, "v0.1.0", got.Version)
		c.AssertExpectations(t)
	}
}

func TestCachedClient_VersionHandlesError(t *testing.T) {
	dir, err := os.MkdirTemp("./", "cache-test")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	c := &MockClient{}
	c.On("Version", "test", "main").Return(mod.Version{}, errors.New("test"))

	cache, err := module.NewCachedClient(c, dir)
	require.NoError(t, err)

	_, err = cache.Version("test", "main")

	assert.Error(t, err)
}

func TestCachedClient_Download(t *testing.T) {
	dir, err := os.MkdirTemp("./", "cache-test")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	r := io.NopCloser(bytes.NewReader([]byte("1234")))
	c := &MockClient{}
	c.On("Download", mod.Version{Path: "test", Version: "main"}).Once().Return(r, nil)

	cache, err := module.NewCachedClient(c, dir)
	require.NoError(t, err)

	got, err := cache.Download(mod.Version{Path: "test", Version: "main"})

	if assert.NoError(t, err) {
		if !assert.Implements(t, (*io.ReadCloser)(nil), got) {
			return
		}
		data, _ := io.ReadAll(got)
		got.Close()
		assert.Equal(t, "1234", string(data))
	}

	// Get from the cache the second time
	got, err = cache.Download(mod.Version{Path: "test", Version: "main"})

	if assert.NoError(t, err) {
		if !assert.Implements(t, (*io.ReadCloser)(nil), got) {
			return
		}
		data, _ := io.ReadAll(got)
		got.Close()
		assert.Equal(t, "1234", string(data))
	}
	c.AssertExpectations(t)
}
