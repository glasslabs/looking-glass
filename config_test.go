package glass_test

import (
	"testing"

	glass "github.com/glasslabs/looking-glass"
	"github.com/glasslabs/looking-glass/module"
	"github.com/stretchr/testify/assert"
)

func TestParseSecrets(t *testing.T) {
	tests := []struct {
		name    string
		in      []byte
		wantErr bool
		want    map[string]interface{}
	}{
		{
			name: "valid config",
			in:   []byte("test:\n  something: 1"),
			want: map[string]interface{}{"test": map[string]interface{}{"something": 1}},
		},
		{
			name:    "invalid config",
			in:      []byte("test: something: 1"),
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := glass.ParseSecrets(test.in)

			if test.wantErr {
				assert.Error(t, err)
				return
			}
			if assert.NoError(t, err) {
				assert.Equal(t, test.want, got)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  glass.Config
		wantErr string
	}{
		{
			name: "valid config",
			config: glass.Config{
				UI: glass.UIConfig{
					Width:  1,
					Height: 1,
				},
				Modules: []module.Descriptor{
					{
						Name: "test-module",
						Path: "test",
					},
				},
			},
			wantErr: "",
		},
		{
			name: "handles zero width",
			config: glass.Config{
				UI: glass.UIConfig{
					Width:  0,
					Height: 1,
				},
				Modules: []module.Descriptor{
					{
						Name: "test-module",
						Path: "test",
					},
				},
			},
			wantErr: "config: ui width and height muse be greater than zero",
		},
		{
			name: "handles zero height",
			config: glass.Config{
				UI: glass.UIConfig{
					Width:  1,
					Height: 0,
				},
				Modules: []module.Descriptor{
					{
						Name: "test-module",
						Path: "test",
					},
				},
			},
			wantErr: "config: ui width and height muse be greater than zero",
		},
		{
			name: "handles no modules",
			config: glass.Config{
				UI: glass.UIConfig{
					Width:  1,
					Height: 1,
				},
			},
			wantErr: "config: at least one module is required",
		},
		{
			name: "handles invalid module",
			config: glass.Config{
				UI: glass.UIConfig{
					Width:  1,
					Height: 1,
				},
				Modules: []module.Descriptor{
					{
						Name: "",
						Path: "test",
					},
				},
			},
			wantErr: "config: a module must have a name",
		},
		{
			name: "handles duplicate module name",
			config: glass.Config{
				UI: glass.UIConfig{
					Width:  1,
					Height: 1,
				},
				Modules: []module.Descriptor{
					{
						Name: "test-module",
						Path: "test",
					},
					{
						Name: "test-module",
						Path: "test1",
					},
				},
			},
			wantErr: "config: module name \"test-module\" is a duplicate. module names must be unique",
		},
		{
			name: "handles mismatched module versions",
			config: glass.Config{
				UI: glass.UIConfig{
					Width:  1,
					Height: 1,
				},
				Modules: []module.Descriptor{
					{
						Name:    "test-module1",
						Path:    "test",
						Version: "1",
					},
					{
						Name:    "test-module2",
						Path:    "test",
						Version: "2",
					},
				},
			},
			wantErr: "config: module \"test\" has mismatched versions (2 != 1)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.Validate()

			if test.wantErr != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, test.wantErr)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name    string
		in      []byte
		secrets map[string]interface{}
		wantErr bool
		want    glass.Config
	}{
		{
			name: "valid config",
			in: []byte(`
ui:
  width: 1024
  height: 768
  fullscreen: false
modules:
  - name: test-mod
    path: some/path
    position: top:right
`),
			want: glass.Config{
				UI: glass.UIConfig{
					Width:      1024,
					Height:     768,
					Fullscreen: false,
				},
				Modules: []module.Descriptor{
					{
						Name:     "test-mod",
						Path:     "some/path",
						Position: module.Position{Vertical: module.Top, Horizontal: module.Right},
					},
				},
			},
		},
		{
			name: "valid config with secrets",
			in: []byte(`
ui:
  width: 1024
  height: 768
  fullscreen: false
modules:
  - name: test-mod
    path: {{ .Secrets.test }}
    position: top:right
`),
			secrets: map[string]interface{}{"test": "some/path"},
			want: glass.Config{
				UI: glass.UIConfig{
					Width:      1024,
					Height:     768,
					Fullscreen: false,
				},
				Modules: []module.Descriptor{
					{
						Name:     "test-mod",
						Path:     "some/path",
						Position: module.Position{Vertical: module.Top, Horizontal: module.Right},
					},
				},
			},
		},
		{
			name:    "invalid config",
			in:      []byte("test: something: 1"),
			wantErr: true,
		},
		{
			name: "invalid config template",
			in: []byte(`
ui:
  width: 1024
  height: 768
  fullscreen: false
modules:
  - name: test-mod
    path: {{ .Secrets.test
    position: top:right
`),
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := glass.ParseConfig(test.in, test.secrets)

			if test.wantErr {
				assert.Error(t, err)
				return
			}
			if assert.NoError(t, err) {
				assert.Equal(t, test.want, got)
			}
		})
	}
}
