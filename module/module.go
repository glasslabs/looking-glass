package module

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/glasslabs/client-go"
	"github.com/hamba/logger/v2"
	lctx "github.com/hamba/logger/v2/ctx"
)

// Module positions.
const (
	Top    = "top"
	Middle = "middle"
	Bottom = "bottom"

	Left   = "left"
	Center = "center"
	Right  = "right"
)

// Position is a module position in the grid.
type Position struct {
	Vertical   string
	Horizontal string
}

// UnmarshalYAML unmarshals a Position from YAML.
func (p *Position) UnmarshalYAML(unmarshal func(any) error) error {
	var pos string
	if err := unmarshal(&pos); err != nil {
		return err
	}

	parts := strings.Split(pos, ":")
	if len(parts) != 2 {
		return errors.New("invalid position: " + pos)
	}

	switch parts[0] {
	case Top:
		p.Vertical = Top
	case Middle:
		p.Vertical = Middle
	case Bottom:
		p.Vertical = Bottom
	default:
		return errors.New("invalid vertical position: " + parts[0])
	}

	switch parts[1] {
	case Left:
		p.Horizontal = Left
	case Center:
		p.Horizontal = Center
	case Right:
		p.Horizontal = Right
	default:
		return errors.New("invalid horizontal position: " + parts[1])
	}
	return nil
}

var modNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)

// Descriptor describes the module and its configuration.
type Descriptor struct {
	Name     string         `yaml:"name"`
	URI      string         `yaml:"uri"`
	Position Position       `yaml:"position"`
	Config   map[string]any `yaml:"config"`
}

// Validate validates a module descriptor.
func (d Descriptor) Validate() error {
	if d.Name == "" {
		return errors.New("config: a module must have a name")
	}
	if !modNameRegex.MatchString(d.Name) {
		return fmt.Errorf("%s: module names may only contain letters, numbers, '-' and '_'", d.Name)
	}

	if d.URI == "" {
		return fmt.Errorf("%s: module must have a URI", d.Name)
	}
	if _, err := url.Parse(d.URI); err != nil {
		return fmt.Errorf("%s: module URI has an error: %w", d.Name, err)
	}

	return nil
}

// WidgetUpdater delivers serialized widget trees to the render pipeline.
type WidgetUpdater interface {
	Update(w client.Widget) error
}

// UIProvider creates module containers and routes widget tree updates.
type UIProvider interface {
	// CreateModule registers a new module container at the given grid position.
	CreateModule(name, vert, horiz string) error
	// ModuleUI returns the WidgetUpdater for the named module.
	// Returns nil if the module has not been registered.
	ModuleUI(name string) WidgetUpdater
}

// PluginInstance is a running WASM plugin driven by the host.
type PluginInstance interface {
	// Run is a long-running call that drives the plugin until ctx is canceled.
	// Plugins manage their own update cadence (polling timers, event subscriptions,
	// etc.) inside Run.
	Run(ctx context.Context) error
	// Close releases plugin resources. Close is called after Run returns.
	Close(ctx context.Context) error
}

// Runner compiles and instantiates WASM plugin modules.
type Runner interface {
	Load(ctx context.Context, name string, wasmBytes []byte, cfg map[string]any) (PluginInstance, error)
	Close(ctx context.Context) error
}

// ExecContext contains context for module execution.
type ExecContext struct {
	AssetsPath string
}

// Loader loads and drives modules.
type Loader struct {
	ui     UIProvider
	d      *Downloader
	runner Runner
	log    *logger.Logger
}

// New returns a module Loader backed by a wazero Runner.
func New(ctx context.Context, ui UIProvider, d *Downloader, execCtx ExecContext, log *logger.Logger) (*Loader, error) {
	runner, err := newWazeroRunner(ctx, ui, execCtx.AssetsPath, log)
	if err != nil {
		return nil, fmt.Errorf("could not create runner: %w", err)
	}

	return &Loader{
		ui:     ui,
		d:      d,
		runner: runner,
		log:    log,
	}, nil
}

// NewWithRunner returns a module Loader using the supplied Runner.
// Intended for testing; production code should use New.
func NewWithRunner(ui UIProvider, d *Downloader, runner Runner, log *logger.Logger) (*Loader, error) {
	return &Loader{
		ui:     ui,
		d:      d,
		runner: runner,
		log:    log,
	}, nil
}

// Load downloads, registers, and starts a module described by desc.
// Setup is called once with the module configuration, then Run is started in a
// goroutine where the plugin manages its own update cadence until ctx is cancelled.
func (l *Loader) Load(ctx context.Context, desc Descriptor) {
	name := strings.ReplaceAll(desc.Name, " ", "_")
	pos := desc.Position

	log := l.log.With(lctx.Str("module", name))

	log.Debug("Downloading module bytes", lctx.Str("uri", desc.URI))

	wasmBytes, err := l.d.DownloadBytes(ctx, desc.URI)
	if err != nil {
		l.log.Error("Could not read module", lctx.Err(err))
		return
	}

	log.Debug("Module downloaded", lctx.Int("bytes", len(wasmBytes)))

	if err = l.ui.CreateModule(name, pos.Vertical, pos.Horizontal); err != nil {
		log.Error("Could not create module", lctx.Err(err))
		return
	}

	log.Debug("Module created", lctx.Str("module", name))

	go func() {
		instance, err := l.runner.Load(ctx, name, wasmBytes, desc.Config)
		if err != nil {
			log.Error("Could not load module", lctx.Err(err))
			return
		}
		defer func() {
			_ = instance.Close(context.WithoutCancel(ctx))
		}()

		l.log.Info("Starting module", lctx.Str("module", name))

		if runErr := instance.Run(ctx); runErr != nil && ctx.Err() == nil {
			l.log.Error("Plugin run error", lctx.Str("module", name), lctx.Err(runErr))
		}
	}()
}

// Close closes the Loader and releases its underlying runner resources.
func (l *Loader) Close(ctx context.Context) error {
	return l.runner.Close(ctx)
}
