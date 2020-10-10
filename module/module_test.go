package module_test

import (
	"context"
	"testing"

	"github.com/glasslabs/looking-glass/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
			if assert.NoError(t, err) {
				assert.Equal(t, test.want.Vertical, got.Vertical)
				assert.Equal(t, test.want.Horizontal, got.Horizontal)
			}
		})
	}
}

func TestBuilder_Build(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		pkg     string
		wantErr bool
	}{
		{
			name:    "valid module",
			path:    "valid",
			wantErr: false,
		},
		{
			name:    "can determine package name",
			path:    "package-name",
			wantErr: false,
		},
		{
			name:    "can receive package name",
			path:    "given-package-name",
			pkg:     "something",
			wantErr: false,
		},
		{
			name:    "handles invalid path",
			path:    "this-path-should-not-exist",
			wantErr: true,
		},
		{
			name:    "handles no NewConfig",
			path:    "no-config",
			wantErr: true,
		},
		{
			name:    "handles no New",
			path:    "no-new",
			pkg:     "new_func",
			wantErr: true,
		},
		{
			name:    "handles bad return",
			path:    "bad-return",
			pkg:     "bad_return",
			wantErr: true,
		},
		{
			name:    "handles module error",
			path:    "mod-error",
			pkg:     "mod_error",
			wantErr: true,
		},
		{
			name:    "handles nil module",
			path:    "mod-nil",
			pkg:     "mod_nil",
			wantErr: true,
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
			ui := &MockUI{}
			log := &MockLogger{}

			bldr, err := module.NewBuilder("testdata")
			require.NoError(t, err)

			mod, err := bldr.Build(context.Background(), desc, ui, log)

			if test.wantErr {
				assert.Error(t, err)
				return
			}
			if assert.NoError(t, err) {
				assert.NotNil(t, mod)
			}
		})
	}
}

type MockUI struct {
	mock.Mock
}

func (m *MockUI) LoadCSS(css string) error {
	args := m.Called(css)
	return args.Error(0)
}

func (m *MockUI) LoadHTML(html string) error {
	args := m.Called(html)
	return args.Error(0)
}

func (m *MockUI) Bind(name string, fun interface{}) error {
	args := m.Called(name, fun)
	return args.Error(0)
}

func (m *MockUI) Eval(cmd string, ctx ...interface{}) (interface{}, error) {
	params := append([]interface{}{cmd}, ctx...)
	args := m.Called(params...)
	return args.Get(0), args.Error(0)
}

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Info(msg string, ctx ...interface{}) {
	params := append([]interface{}{msg}, ctx...)
	_ = m.Called(params)
}

func (m *MockLogger) Error(msg string, ctx ...interface{}) {
	params := append([]interface{}{msg}, ctx...)
	_ = m.Called(params)
}
