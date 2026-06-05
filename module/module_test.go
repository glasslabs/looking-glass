package module_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/glasslabs/looking-glass/module"
	"github.com/hamba/logger/v2"
	"github.com/hamba/testutils/retry"
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
			name:    "valid descriptor",
			desc:    module.Descriptor{Name: "test-module", URI: "test"},
			wantErr: "",
		},
		{
			name:    "handles no name",
			desc:    module.Descriptor{Name: "", URI: "test"},
			wantErr: "config: a module must have a name",
		},
		{
			name:    "handles invalid name",
			desc:    module.Descriptor{Name: "test@modile", URI: "test"},
			wantErr: "test@modile: module names may only contain letters, numbers, '-' and '_'",
		},
		{
			name:    "handles no path",
			desc:    module.Descriptor{Name: "test-module", URI: ""},
			wantErr: "test-module: module must have a URI",
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

func TestLoader_Load(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.New(&buf, logger.LogfmtFormat(), logger.Info)

	ui := &mockUIProvider{}
	ui.On("CreateModule", "test", "top", "right").Once().Return(nil)

	inst := &mockPluginInstance{}
	inst.On("Run", mock.Anything).Maybe().Return(nil)
	inst.On("Close", mock.Anything).Maybe().Return(nil)

	runner := &mockRunner{}
	runner.On("Load", mock.Anything, "test", mock.AnythingOfType("[]uint8"), mock.Anything).
		Once().Return(inst, nil)

	d, err := module.NewDownloader("./testdata", log)
	require.NoError(t, err)

	loader, err := module.NewWithRunner(ui, d, runner, log)
	require.NoError(t, err)

	desc := module.Descriptor{
		Name:     "test",
		URI:      "minimal.wasm",
		Position: module.Position{Vertical: module.Top, Horizontal: module.Right},
		Config:   map[string]any{"a": "b"},
	}
	loader.Load(t.Context(), desc)

	retry.Run(t, func(t *retry.SubT) {
		ui.AssertExpectations(t)
		inst.AssertExpectations(t)
		runner.AssertExpectations(t)
	})
}

func TestLoader_LoadWithWazeroIntegration(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.New(&buf, logger.LogfmtFormat(), logger.Info)

	ui := &mockUIProvider{}
	ui.On("CreateModule", "test", "top", "right").Once().Return(nil)

	d, err := module.NewDownloader("./testdata", log)
	require.NoError(t, err)

	loader, err := module.New(t.Context(), ui, d, module.ExecContext{}, log)
	require.NoError(t, err)

	desc := module.Descriptor{
		Name:     "test",
		URI:      "minimal.wasm",
		Position: module.Position{Vertical: module.Top, Horizontal: module.Right},
	}
	loader.Load(t.Context(), desc)

	retry.Run(t, func(t *retry.SubT) {
		ui.AssertExpectations(t)
	})
}

type mockUIProvider struct{ mock.Mock }

func (m *mockUIProvider) CreateModule(name, vert, horiz string) {
	m.Called(name, vert, horiz)
}

func (m *mockUIProvider) ModuleUI(name string) module.WidgetUpdater {
	args := m.Called(name)
	if v := args.Get(0); v != nil {
		return v.(module.WidgetUpdater)
	}
	return nil
}

type mockRunner struct{ mock.Mock }

func (m *mockRunner) Load(ctx context.Context, name string, wasmBytes []byte, cfg map[string]any) (module.PluginInstance, error) {
	args := m.Called(ctx, name, wasmBytes, cfg)
	return args.Get(0).(module.PluginInstance), args.Error(1)
}

func (m *mockRunner) Close(context.Context) error { return nil }

type mockPluginInstance struct{ mock.Mock }

func (m *mockPluginInstance) Setup(ctx context.Context, cfg map[string]any) error {
	return m.Called(ctx, cfg).Error(0)
}

func (m *mockPluginInstance) Run(ctx context.Context) error   { return m.Called(ctx).Error(0) }
func (m *mockPluginInstance) Close(ctx context.Context) error { return m.Called(ctx).Error(0) }
