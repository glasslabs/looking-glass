package module

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/hamba/logger/v2"
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

// UI represents the UI manager.
type UI interface {
	Eval(js string) (any, error)
}

// Runner represents a WASM runner.
type Runner interface {
	Run(name, path string, cfg map[string]any) error
}

// ExecContext contains context for module execution.
type ExecContext struct {
	ModuleURL string
	AssetsURL string
}

// Loader loads modules.
type Loader struct {
	ui     UI
	d      *Downloader
	modURL *url.URL
	env    map[string]string

	runner Runner

	log *logger.Logger
}

// New returns a module loader.
func New(ui UI, d *Downloader, execCtx ExecContext, log *logger.Logger) (*Loader, error) {
	u, err := url.Parse(execCtx.ModuleURL)
	if err != nil {
		return nil, fmt.Errorf("parsing modURL: %w", err)
	}

	env := map[string]string{
		"ASSETS_URL": execCtx.AssetsURL,
	}

	runner, err := NewGoRunner(ui, env)
	if err != nil {
		return nil, fmt.Errorf("creating go module runner: %w", err)
	}

	return &Loader{
		ui:     ui,
		d:      d,
		modURL: u,
		env:    env,
		runner: runner,
		log:    log,
	}, nil
}

// Load loads a module.
func (l *Loader) Load(ctx context.Context, desc Descriptor) error {
	path, err := l.d.Download(ctx, desc.URI)
	if err != nil {
		return fmt.Errorf("reading module: %w", err)
	}

	u := l.modURL.JoinPath(path)

	name := strings.ReplaceAll(desc.Name, " ", "_")
	pos := desc.Position
	if _, err = l.ui.Eval(fmt.Sprintf(`createModule(%q, %q, %q);`, name, pos.Vertical, pos.Horizontal)); err != nil {
		return fmt.Errorf("%s: creating module ui element: %w", name, err)
	}

	if err = l.runner.Run(name, u.String(), desc.Config); err != nil {
		return fmt.Errorf("%s: running module: %w", name, err)
	}
	return nil
}
