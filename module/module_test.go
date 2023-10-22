package module_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/glasslabs/looking-glass/module"
	"github.com/hamba/logger/v2"
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
			name: "valid descriptor",
			desc: module.Descriptor{
				Name: "test-module",
				URI:  "test",
			},
			wantErr: "",
		},
		{
			name: "handles no name",
			desc: module.Descriptor{
				Name: "",
				URI:  "test",
			},
			wantErr: "config: a module must have a name",
		},
		{
			name: "handles invalid name",
			desc: module.Descriptor{
				Name: "test@modile",
				URI:  "test",
			},
			wantErr: "test@modile: module names may only contain letters, numbers, '-' and '_'",
		},
		{
			name: "handles no path",
			desc: module.Descriptor{
				Name: "test-module",
				URI:  "",
			},
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

func TestModule_Load(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(&buf, logger.LogfmtFormat(), logger.Info)

	ui := &mockUI{}
	ui.On("Eval", `createModule("test", "top", "right");`).
		Once().
		Return(nil, nil)
	ui.On("Eval", mock.AnythingOfType("string")).
		Once().
		Return(nil, nil)

	d, err := module.NewDownloader("./testdata", log)
	require.NoError(t, err)

	mod, err := module.New(ui, d, module.ExecContext{}, log)
	require.NoError(t, err)

	desc := module.Descriptor{
		Name: "test",
		URI:  "test.wasm",
		Position: module.Position{
			Vertical:   module.Top,
			Horizontal: module.Right,
		},
		Config: map[string]any{
			"a": "b",
		},
	}
	err = mod.Load(context.Background(), desc)

	require.NoError(t, err)
	ui.AssertExpectations(t)
}

type mockUI struct {
	mock.Mock
}

func (m *mockUI) Eval(js string) (any, error) {
	args := m.Called(js)
	return args.Get(0), args.Error(1)
}
