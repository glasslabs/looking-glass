package module_test

import (
	"testing"

	"github.com/glasslabs/looking-glass/module"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGoRunner_Run(t *testing.T) {
	ui := &mockUI{}
	ui.On("Eval", mock.AnythingOfType("string")).
		Once().
		Return(nil, nil)

	env := map[string]string{
		"FOO": "bar",
	}

	r, err := module.NewGoRunner(ui, env)
	require.NoError(t, err)

	err = r.Run("test-mod", "my-url", map[string]any{"a": "b"})

	require.NoError(t, err)
	ui.AssertExpectations(t)
}
