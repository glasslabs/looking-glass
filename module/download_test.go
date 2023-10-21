package module_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/glasslabs/looking-glass/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownload_DownloadLocalFile(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		want    string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:    "reads a file",
			uri:     "file:///test.wasm",
			want:    "/test.wasm",
			wantErr: require.NoError,
		},
		{
			name:    "reads a file with no scheme",
			uri:     "test.wasm",
			want:    "test.wasm",
			wantErr: require.NoError,
		},
		{
			name:    "handles no file",
			uri:     "file:///doesnotexist.wasm",
			wantErr: require.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			r, err := module.NewDownloader("./testdata")
			require.NoError(t, err)

			got, err := r.Download(context.Background(), test.uri)

			test.wantErr(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestDownload_DownloadHTTPFile(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:    "reads http files",
			path:    "/testdata/test.wasm",
			want:    "/testdata/test.wasm",
			wantErr: require.NoError,
		},
		{
			name:    "handles no file",
			path:    "/testdata/doesnotexist.wasm",
			wantErr: require.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if req.URL.Path != "/testdata/test.wasm" {
					rw.WriteHeader(http.StatusNotFound)
					return
				}

				b, err := os.ReadFile("testdata/test.wasm")
				assert.NoError(t, err)
				_, _ = rw.Write(b)
			}))
			t.Cleanup(srv.Close)

			tmpDir := t.TempDir()

			r, err := module.NewDownloader(tmpDir)
			require.NoError(t, err)

			url := srv.URL + test.path
			got, err := r.Download(context.Background(), url)

			test.wantErr(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestDownload_DownloadHTTPCachedFile(t *testing.T) {
	var called int
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		called++

		b, err := os.ReadFile("testdata/test.wasm")
		assert.NoError(t, err)
		_, _ = rw.Write(b)
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()

	r, err := module.NewDownloader(tmpDir)
	require.NoError(t, err)

	url := srv.URL + "/testdata/test.wasm"
	_, err = r.Download(context.Background(), url)
	require.NoError(t, err)

	_, err = r.Download(context.Background(), url)

	require.NoError(t, err)
	assert.Equal(t, 1, called)
}

func requireFile(t *testing.T, path string) []byte {
	t.Helper()

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return b
}
