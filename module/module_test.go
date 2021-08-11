package module_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/glasslabs/looking-glass/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mod "golang.org/x/mod/module"
	"gopkg.in/yaml.v3"
)

func TestPosition_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		position string
		want     module.Position
		wantErr  string
	}{
		{
			position: "top:left",
			want:     module.Position{Vertical: module.Top, Horizontal: module.Left},
		},
		{
			position: "middle:left",
			want:     module.Position{Vertical: module.Middle, Horizontal: module.Left},
		},
		{
			position: "bottom:left",
			want:     module.Position{Vertical: module.Bottom, Horizontal: module.Left},
		},
		{
			position: "top:center",
			want:     module.Position{Vertical: module.Top, Horizontal: module.Center},
		},
		{
			position: "top:right",
			want:     module.Position{Vertical: module.Top, Horizontal: module.Right},
		},
		{
			position: "something:left",
			wantErr:  "invalid vertical position: something",
		},
		{
			position: "top:something",
			wantErr:  "invalid horizontal position: something",
		},
		{
			position: "top::left",
			wantErr:  "invalid position: top::left",
		},
	}

	for _, test := range tests {
		t.Run(test.position, func(t *testing.T) {
			var got module.Position
			err := yaml.Unmarshal([]byte(test.position), &got)

			if test.wantErr != "" {
				assert.EqualError(t, err, test.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want.Vertical, got.Vertical)
			assert.Equal(t, test.want.Horizontal, got.Horizontal)
		})
	}
}

func TestDescriptor_Validate(t *testing.T) {
	tests := []struct {
		name    string
		desc    module.Descriptor
		wantErr string
	}{
		{
			name: "valid descriptor",
			desc: module.Descriptor{
				Name: "test-module",
				Path: "test",
			},
			wantErr: "",
		},
		{
			name: "handles no name",
			desc: module.Descriptor{
				Name: "",
				Path: "test",
			},
			wantErr: "config: a module must have a name",
		},
		{
			name: "handles invalid name",
			desc: module.Descriptor{
				Name: "test@modile",
				Path: "test",
			},
			wantErr: "test@modile: module names may only contain letters, numbers, '-' and '_'",
		},
		{
			name: "handles no path",
			desc: module.Descriptor{
				Name: "test-module",
				Path: "",
			},
			wantErr: "test-module: module must have a path",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.desc.Validate()

			if test.wantErr != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, test.wantErr)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestService_Extract(t *testing.T) {
	dir, err := os.MkdirTemp("./", "extract-test")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	b, err := os.ReadFile("../testdata/module@v0.1.0.zip")
	require.NoError(t, err)
	r1 := io.NopCloser(bytes.NewReader(b))
	b, err = os.ReadFile("../testdata/module@v0.2.0.zip")
	require.NoError(t, err)
	r2 := io.NopCloser(bytes.NewReader(b))
	c := &MockClient{}
	c.On("Version", "test-module", "main").Twice().Return(mod.Version{Path: "test-module", Version: "v0.1.0"}, nil)
	c.On("Version", "test-module", "latest").Once().Return(mod.Version{Path: "test-module", Version: "v0.2.0"}, nil)
	c.On("Download", mod.Version{Path: "test-module", Version: "v0.1.0"}).Once().Return(r1, nil)
	c.On("Download", mod.Version{Path: "test-module", Version: "v0.2.0"}).Once().Return(r2, nil)

	svc, err := module.NewService(dir, c)
	require.NoError(t, err)

	err = svc.Extract(module.Descriptor{
		Name:    "test",
		Path:    "test-module",
		Version: "main",
	})

	if assert.NoError(t, err) {
		b, _ := os.ReadFile(filepath.Join(dir, "src/test-module/main.go"))
		assert.Equal(t, "test-module\n", string(b))
		b, _ = os.ReadFile(filepath.Join(dir, "src/test-module/.looking-glass"))
		assert.Equal(t, "v0.1.0", string(b))
	}

	err = svc.Extract(module.Descriptor{
		Name:    "test",
		Path:    "test-module",
		Version: "main",
	})

	if assert.NoError(t, err) {
		b, _ = os.ReadFile(filepath.Join(dir, "src/test-module/main.go"))
		assert.Equal(t, "test-module\n", string(b))
		b, _ = os.ReadFile(filepath.Join(dir, "src/test-module/.looking-glass"))
		assert.Equal(t, "v0.1.0", string(b))
	}

	err = svc.Extract(module.Descriptor{
		Name:    "test",
		Path:    "test-module",
		Version: "latest",
	})

	require.NoError(t, err)
	b, _ = os.ReadFile(filepath.Join(dir, "src/test-module/main.go"))
	assert.Equal(t, "test-module\n", string(b))
	b, _ = os.ReadFile(filepath.Join(dir, "src/test-module/.looking-glass"))
	assert.Equal(t, "v0.2.0", string(b))
}

func TestService_ExtractWithVendor(t *testing.T) {
	dir, err := os.MkdirTemp("./", "extract-test")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	b, err := os.ReadFile("../testdata/outer-module@v0.1.0.zip")
	require.NoError(t, err)
	r1 := io.NopCloser(bytes.NewReader(b))
	b, err = os.ReadFile("../testdata/module@v0.2.0.zip")
	require.NoError(t, err)
	r2 := io.NopCloser(bytes.NewReader(b))
	c := &MockClient{}
	c.On("Version", "outer-module", "v0.1.0").Twice().Return(mod.Version{Path: "outer-module", Version: "v0.1.0"}, nil)
	c.On("Version", "test-module", "v0.2.0").Once().Return(mod.Version{Path: "test-module", Version: "v0.2.0"}, nil)
	c.On("Download", mod.Version{Path: "outer-module", Version: "v0.1.0"}).Once().Return(r1, nil)
	c.On("Download", mod.Version{Path: "test-module", Version: "v0.2.0"}).Once().Return(r2, nil)

	svc, err := module.NewService(dir, c)
	require.NoError(t, err)

	err = svc.Extract(module.Descriptor{
		Name:    "test",
		Path:    "outer-module",
		Version: "v0.1.0",
	})
	require.NoError(t, err)

	b, _ = os.ReadFile(filepath.Join(dir, "src/test-module/main.go"))
	assert.Equal(t, "test-module\n", string(b))
	b, _ = os.ReadFile(filepath.Join(dir, "src/test-module/.looking-glass"))
	assert.Equal(t, "v0.2.0", string(b))
}

func TestService_ExtractLeavesUserModule(t *testing.T) {
	dir, err := os.MkdirTemp("./", "extract-test")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	err = os.MkdirAll(filepath.Join(dir, "src/test-module"), 0777)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "src/test-module/main.go"), []byte("something"), 0544)
	require.NoError(t, err)

	c := &MockClient{}
	c.On("Version", "test-module", "main").Return(mod.Version{Path: "test-module", Version: "v0.1.0"}, nil)

	svc, err := module.NewService(dir, c)
	require.NoError(t, err)

	err = svc.Extract(module.Descriptor{
		Name:    "test",
		Path:    "test-module",
		Version: "main",
	})

	require.NoError(t, err)
	b, _ := os.ReadFile(filepath.Join(dir, "src/test-module/main.go"))
	assert.Equal(t, "something", string(b))
	_, err = os.Stat(filepath.Join(dir, "src/test-module/.looking-glass"))
	assert.Error(t, err)
	c.AssertExpectations(t)
}

func TestService_Run(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		pkg     string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:    "valid module",
			path:    "valid",
			wantErr: require.NoError,
		},
		{
			name:    "can determine package name",
			path:    "package-name",
			wantErr: require.NoError,
		},
		{
			name:    "can receive package name",
			path:    "given-package-name",
			pkg:     "something",
			wantErr: require.NoError,
		},
		{
			name:    "handles invalid path",
			path:    "this-path-should-not-exist",
			wantErr: require.Error,
		},
		{
			name:    "handles no NewConfig",
			path:    "no-config",
			wantErr: require.Error,
		},
		{
			name:    "handles no New",
			path:    "no-new",
			pkg:     "new_func",
			wantErr: require.Error,
		},
		{
			name:    "handles bad return",
			path:    "bad-return",
			pkg:     "bad_return",
			wantErr: require.Error,
		},
		{
			name:    "handles module error",
			path:    "mod-error",
			pkg:     "mod_error",
			wantErr: require.Error,
		},
		{
			name:    "handles nil module",
			path:    "mod-nil",
			pkg:     "mod_nil",
			wantErr: require.Error,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			desc := module.Descriptor{
				Name:    "test",
				Path:    test.path,
				Package: test.pkg,
				Config:  yaml.Node{},
			}
			client := &MockClient{}
			ui := &MockUI{}
			log := &MockLogger{}

			svc, err := module.NewService("../testdata/mod", client)
			require.NoError(t, err)

			mod, err := svc.Run(context.Background(), desc, ui, log)

			test.wantErr(t, err)
			if mod != nil {
				assert.Implements(t, (*io.Closer)(nil), mod)
			}
		})
	}
}
