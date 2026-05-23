package module

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/hamba/logger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero/api"
)

func TestNewWazeroRunner(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.New(&buf, logger.LogfmtFormat(), logger.Info)

	runner, err := newWazeroRunner(t.Context(), noopUI{}, "", log)

	require.NoError(t, err)
	require.NotNil(t, runner)
	t.Cleanup(func() { _ = runner.Close(context.Background()) })
}

func TestWazeroRunner_Load(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.New(&buf, logger.LogfmtFormat(), logger.Info)

	runner, err := newWazeroRunner(t.Context(), noopUI{}, "", log)
	require.NoError(t, err)
	t.Cleanup(func() { _ = runner.Close(context.Background()) })

	wasmBytes, err := os.ReadFile("./testdata/minimal.wasm")
	require.NoError(t, err)

	inst, err := runner.Load(t.Context(), "test", wasmBytes, nil)

	require.NoError(t, err)
	require.NotNil(t, inst)
	t.Cleanup(func() { _ = inst.Close(context.Background()) })
}

func TestWazeroRunner_LoadCachesCompiledModule(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.New(&buf, logger.LogfmtFormat(), logger.Info)

	runner, err := newWazeroRunner(t.Context(), noopUI{}, "", log)
	require.NoError(t, err)
	t.Cleanup(func() { _ = runner.Close(context.Background()) })

	wasmBytes, err := os.ReadFile("./testdata/minimal.wasm")
	require.NoError(t, err)

	inst1, err := runner.Load(t.Context(), "first", wasmBytes, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = inst1.Close(context.Background()) })

	inst2, err := runner.Load(t.Context(), "second", wasmBytes, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = inst2.Close(context.Background()) })

	assert.Contains(t, buf.String(), "Using cached compiled module")
}

func TestWazeroRunner_LoadHandlesInvalidWasm(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.New(&buf, logger.LogfmtFormat(), logger.Info)

	runner, err := newWazeroRunner(t.Context(), noopUI{}, "", log)
	require.NoError(t, err)
	t.Cleanup(func() { _ = runner.Close(context.Background()) })

	_, err = runner.Load(t.Context(), "bad", []byte("not-wasm"), nil)

	require.Error(t, err)
	assert.ErrorContains(t, err, "compiling module bad")
}

func TestWazeroRunner_LoadHandlesInvalidConfig(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.New(&buf, logger.LogfmtFormat(), logger.Info)

	runner, err := newWazeroRunner(t.Context(), noopUI{}, "", log)
	require.NoError(t, err)
	t.Cleanup(func() { _ = runner.Close(context.Background()) })

	cfg := map[string]any{"key": make(chan int)}

	_, err = runner.Load(t.Context(), "test", []byte("any"), cfg)

	require.Error(t, err)
	assert.ErrorContains(t, err, "encoding config for test")
}

func TestWazeroRunner_Close(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.New(&buf, logger.LogfmtFormat(), logger.Info)

	runner, err := newWazeroRunner(t.Context(), noopUI{}, "", log)
	require.NoError(t, err)

	err = runner.Close(context.Background())

	require.NoError(t, err)
}

func TestWazeroInstance_RunNoStartFunction(t *testing.T) {
	t.Parallel()

	mod := &stubModule{fn: nil}
	inst := &wazeroInstance{mod: mod, name: "test"}

	err := inst.Run(t.Context())

	require.NoError(t, err)
}

func TestWazeroInstance_RunStartReturnsError(t *testing.T) {
	t.Parallel()

	fn := &stubFunction{err: errors.New("start failed")}
	mod := &stubModule{fn: fn}
	inst := &wazeroInstance{mod: mod, name: "test"}

	err := inst.Run(t.Context())

	require.Error(t, err)
	assert.ErrorContains(t, err, "run: start failed")
}

func TestWazeroInstance_RunHandlesContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fn := &stubFunction{err: errors.New("interrupted by wazero")}
	mod := &stubModule{fn: fn}
	inst := &wazeroInstance{mod: mod, name: "test"}

	err := inst.Run(ctx)

	require.NoError(t, err)
}

func TestWazeroInstance_Close(t *testing.T) {
	t.Parallel()

	closed := false
	mod := &stubModule{
		closeFn: func(_ context.Context) error {
			closed = true
			return nil
		},
	}
	inst := &wazeroInstance{mod: mod, name: "test"}

	err := inst.Close(t.Context())

	require.NoError(t, err)
	assert.True(t, closed)
}

func TestWazeroInstance_RunIntegration(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.New(&buf, logger.LogfmtFormat(), logger.Info)

	runner, err := newWazeroRunner(t.Context(), noopUI{}, "", log)
	require.NoError(t, err)
	t.Cleanup(func() { _ = runner.Close(context.Background()) })

	wasmBytes, err := os.ReadFile("./testdata/minimal.wasm")
	require.NoError(t, err)

	inst, err := runner.Load(t.Context(), "test", wasmBytes, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = inst.Close(context.Background()) })

	err = inst.Run(t.Context())

	require.NoError(t, err)
}

func TestWazeroInstance_RunIntegrationHandlesContextCancelled(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.New(&buf, logger.LogfmtFormat(), logger.Info)

	runner, err := newWazeroRunner(t.Context(), noopUI{}, "", log)
	require.NoError(t, err)
	t.Cleanup(func() { _ = runner.Close(context.Background()) })

	wasmBytes, err := os.ReadFile("./testdata/minimal.wasm")
	require.NoError(t, err)

	inst, err := runner.Load(t.Context(), "test", wasmBytes, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = inst.Close(context.Background()) })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = inst.Run(ctx)

	require.NoError(t, err)
}

func TestPluginLogWriter_WriteBuffersAndFlushesLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      []string
		wantOutput string
	}{
		{
			name:       "single line with newline is dispatched",
			input:      []string{"msg=hello\n"},
			wantOutput: "hello",
		},
		{
			name:       "multiple lines in one write are all dispatched",
			input:      []string{"msg=first\nmsg=second\n"},
			wantOutput: "first",
		},
		{
			name:       "partial write without newline is buffered",
			input:      []string{"msg=buffered"},
			wantOutput: "",
		},
		{
			name:       "partial write completed by subsequent write",
			input:      []string{"msg=comp", "leted\n"},
			wantOutput: "completed",
		},
		{
			name:       "empty line is skipped",
			input:      []string{"\n"},
			wantOutput: "",
		},
		{
			name:       "CRLF line endings are handled",
			input:      []string{"msg=crlf\r\n"},
			wantOutput: "crlf",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			log := logger.New(&buf, logger.LogfmtFormat(), logger.Trace)
			w := newPluginLogWriter("test", log)

			for _, chunk := range test.input {
				n, err := w.Write([]byte(chunk))

				require.NoError(t, err)
				assert.Equal(t, len(chunk), n)
			}

			if test.wantOutput != "" {
				assert.Contains(t, buf.String(), test.wantOutput)
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestPluginLogWriter_WriteDispatchesLogLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "error level is dispatched",
			line: "level=error msg=test-error\n",
			want: "lvl=eror msg=test-error module=test\n",
		},
		{
			name: "warn level is dispatched",
			line: "level=warn msg=test-warn\n",
			want: "lvl=warn msg=test-warn module=test\n",
		},
		{
			name: "debug level is dispatched",
			line: "level=debug msg=test-debug\n",
			want: "lvl=dbug msg=test-debug module=test\n",
		},
		{
			name: "default level routes to info",
			line: "level=unknown msg=test-default\n",
			want: "lvl=info msg=test-default module=test\n",
		},
		{
			name: "no level field defaults to info",
			line: "msg=test-nolevel\n",
			want: "lvl=info msg=test-nolevel module=test\n",
		},
		{
			name: "no msg field uses raw line as message",
			line: "level=info raw-line-content\n",
			want: "lvl=info msg=raw-line-content module=test\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			log := logger.New(&buf, logger.LogfmtFormat(), logger.Trace)
			w := newPluginLogWriter("test", log)

			_, err := w.Write([]byte(test.line))

			require.NoError(t, err)
			assert.Equal(t, buf.String(), test.want)
		})
	}
}

func TestParseLogFmt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want map[string]string
	}{
		{
			name: "parses single pair",
			line: "key=val",
			want: map[string]string{"key": "val"},
		},
		{
			name: "parses multiple pairs",
			line: "level=info msg=hello foo=bar",
			want: map[string]string{"level": "info", "msg": "hello", "foo": "bar"},
		},
		{
			name: "parses quoted value preserves spaces",
			line: `msg="hello world" foo=bar`,
			want: map[string]string{"msg": "hello world", "foo": "bar"},
		},
		{
			name: "handles empty line returns no fields",
			line: "",
			want: map[string]string{},
		},
		{
			name: "handles leading and trailing spaces",
			line: "  key=val  ",
			want: map[string]string{"key": "val"},
		},
		{
			name: "handles no ending quotes",
			line: `key=val foo="test`,
			want: map[string]string{"key": "val"},
		},
		{
			name: "handles no equals",
			line: "some text",
			want: map[string]string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := parseLogFmt(test.line)

			assert.Equal(t, test.want, got)
		})
	}
}

type noopUI struct{}

func (noopUI) CreateModule(_, _, _ string) error { return nil }
func (noopUI) ModuleUI(_ string) WidgetUpdater   { return nil }

type stubModule struct {
	api.Module // nil embedding; panics on any unexpected method call
	fn         api.Function
	closeFn    func(context.Context) error
}

func (s *stubModule) ExportedFunction(_ string) api.Function { return s.fn }
func (s *stubModule) Close(ctx context.Context) error {
	if s.closeFn != nil {
		return s.closeFn(ctx)
	}
	return nil
}

type stubFunction struct {
	api.Function // nil embedding
	err          error
}

func (f *stubFunction) Call(_ context.Context, _ ...uint64) ([]uint64, error) {
	return nil, f.err
}
