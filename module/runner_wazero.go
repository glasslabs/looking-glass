package module

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/hamba/logger/v2"
	lctx "github.com/hamba/logger/v2/ctx"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

// newWazeroRunner returns a Runner backed by a shared wazero runtime.
// WASI and the looking-glass host module are instantiated once into the
// runtime; individual plugin instances are compiled and cached by SHA-256.
func newWazeroRunner(ctx context.Context, ui UIProvider, assetsPath string, log *logger.Logger) (*wazeroRunner, error) {
	rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithCloseOnContextDone(true))

	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	if err := buildHostModule(ctx, rt, ui, log); err != nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("building host module: %w", err)
	}

	return &wazeroRunner{
		runtime:    rt,
		assetsPath: assetsPath,
		compiled:   make(map[[32]byte]wazero.CompiledModule),
		log:        log,
	}, nil
}

// wazeroRunner compiles WASM bytes with wazero and returns PluginInstances.
// Compiled modules are cached by SHA-256 of their bytes so the same binary
// loaded at two positions is only compiled once.
type wazeroRunner struct {
	runtime    wazero.Runtime
	assetsPath string

	mu       sync.Mutex
	compiled map[[32]byte]wazero.CompiledModule

	log *logger.Logger
}

// Load compiles (or retrieves from cache) a WASM module and returns a
// PluginInstance ready to drive with Run/Close.
func (r *wazeroRunner) Load(ctx context.Context, name string, wasmBytes []byte, cfg map[string]any) (PluginInstance, error) {
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("encoding config for %s: %w", name, err)
	}

	hash := sha256.Sum256(wasmBytes)

	r.mu.Lock()
	compiled, ok := r.compiled[hash]
	if !ok {
		r.log.Info("Compiling WASM binary", lctx.Str("module", name), lctx.Int("bytes", len(wasmBytes)))
		compiled, err = r.runtime.CompileModule(ctx, wasmBytes)
		if err != nil {
			r.mu.Unlock()
			return nil, fmt.Errorf("compiling module %s: %w", name, err)
		}
		r.log.Info("WASM compilation done", lctx.Str("module", name))
		r.compiled[hash] = compiled
	} else {
		r.log.Info("Using cached compiled module", lctx.Str("module", name))
	}
	r.mu.Unlock()

	modCfg := wazero.NewModuleConfig().
		WithSysWalltime().
		WithSysNanotime().
		WithSysNanosleep().
		WithStartFunctions(). // Suppress auto-call of _start; Run() drives it.
		WithEnv("MODULE_NAME", name).
		WithEnv("MODULE_CONFIG", string(cfgJSON)).
		WithStderr(newPluginLogWriter(name, r.log)).
		WithName(name)

	if r.assetsPath != "" {
		modCfg = modCfg.WithFSConfig(
			wazero.NewFSConfig().WithReadOnlyDirMount(r.assetsPath, "/assets"),
		)
	}

	r.log.Debug("Instantiating module", lctx.Str("module", name))

	mod, err := r.runtime.InstantiateModule(ctx, compiled, modCfg)
	if err != nil {
		return nil, fmt.Errorf("instantiating module %s: %w", name, err)
	}

	r.log.Info("Module instantiated", lctx.Str("module", name))

	return &wazeroInstance{mod: mod, name: name}, nil
}

// Close shuts down the wazero runtime and all instantiated modules.
func (r *wazeroRunner) Close(ctx context.Context) error {
	r.log.Debug("Closing WASM runner")

	return r.runtime.Close(ctx)
}

// wazeroInstance wraps a wazero module instance, driving it via Run/Close.
type wazeroInstance struct {
	mod  api.Module
	name string
}

// Run calls the WASI _start export, which invokes the plugin's main() function.
// main() performs setup and then enters a blocking update loop. Execution
// continues until ctx is canceled, at which point wazero interrupts the call
// and returns a context error (treated as a clean shutdown, not an error).
func (i *wazeroInstance) Run(ctx context.Context) error {
	startFn := i.mod.ExportedFunction("_start")
	if startFn == nil {
		return nil
	}
	_, err := startFn.Call(ctx)
	if err != nil && ctx.Err() == nil {
		if exitErr, ok := errors.AsType[*sys.ExitError](err); ok && exitErr.ExitCode() == 0 {
			return nil
		}
		return fmt.Errorf("run: %w", err)
	}
	return nil
}

// Close releases the module instance.
func (i *wazeroInstance) Close(ctx context.Context) error {
	return i.mod.Close(ctx)
}

// pluginLogWriter is a line-buffered io.Writer that parses logfmt lines
// written by the plugin to stderr and routes them to the host logger.
type pluginLogWriter struct {
	name string
	log  *logger.Logger

	mu  sync.Mutex
	buf []byte
}

// newPluginLogWriter returns a pluginLogWriter for the named plugin.
func newPluginLogWriter(name string, log *logger.Logger) *pluginLogWriter {
	return &pluginLogWriter{name: name, log: log}
}

func (w *pluginLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf = append(w.buf, p...)
	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx < 0 {
			break
		}
		line := strings.TrimRight(string(w.buf[:idx]), "\r")
		w.buf = w.buf[idx+1:]
		if line != "" {
			w.dispatch(line)
		}
	}
	return len(p), nil
}

func (w *pluginLogWriter) dispatch(line string) {
	kvs := parseLogFmt(line)

	level := kvs["level"]
	msg := kvs["msg"]
	if msg == "" {
		if level != "" {
			// We know the level must be first for this to happen, so trim it off if it's there.
			line = strings.TrimPrefix(strings.TrimSpace(line), "level="+level)
		}
		msg = strings.TrimSpace(line)
	}

	fields := make([]logger.Field, 0, len(kvs))
	fields = append(fields, lctx.Str("module", w.name))
	for k, v := range kvs {
		if k == "level" || k == "msg" {
			continue
		}
		fields = append(fields, lctx.Str(k, v))
	}

	switch level {
	case "error":
		w.log.Error(msg, fields...)
	case "warn":
		w.log.Warn(msg, fields...)
	case "debug":
		w.log.Debug(msg, fields...)
	default:
		w.log.Info(msg, fields...)
	}
}

func parseLogFmt(line string) map[string]string {
	fields := make(map[string]string, 3)
	s := strings.TrimSpace(line)
	for s != "" {
		key, rest, found := strings.Cut(s, "=")
		if !found || rest == "" {
			// Either no '=' found or no value after '='. Skip what is left.
			break
		}
		if rest[0] == '"' {
			rest = rest[1:]
			// The value is quoted. Find the other quote to fund the value.
			before, after, ok := strings.Cut(rest, "\"")
			if !ok {
				// No ending quote. Invalid.
				break
			}
			fields[key] = before
			s = strings.TrimPrefix(after, " ")
			continue
		}
		// The string is not quoted, the next space is the end of the value.
		before, after, ok := strings.Cut(rest, " ")
		if !ok {
			fields[key] = rest
			break
		}
		fields[key] = before
		s = strings.TrimSpace(after)
	}
	return fields
}
