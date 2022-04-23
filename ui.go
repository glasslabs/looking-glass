package glass

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/glasslabs/looking-glass/module"
	"github.com/vincent-petithory/dataurl"
	"github.com/zserge/lorca"
)

var (
	//go:embed webui/index.html
	page []byte

	//go:embed webui/fonts.css
	fonts []byte
)

// UIConfig contains configuration for the UI.
type UIConfig struct {
	Width      int      `yaml:"width"`
	Height     int      `yaml:"height"`
	Fullscreen bool     `yaml:"fullscreen"`
	CustomCSS  []string `yaml:"customCss"`
}

// Validate validates the ui configuration.
func (c UIConfig) Validate() error {
	if c.Width <= 0 || c.Height <= 0 {
		return errors.New("config: ui width and height muse be greater than zero")
	}

	return nil
}

// UI implements a ui manager.
type UI struct {
	win lorca.UI
}

// NewUI returns a new UI.
func NewUI(cfg UIConfig) (*UI, error) {
	var args []string
	if cfg.Fullscreen {
		args = append(args, "--start-fullscreen")
	}
	url := dataurl.New(page, "text/html")
	win, err := lorca.New(url.String(), "", cfg.Width, cfg.Height, args...)
	if err != nil {
		return nil, fmt.Errorf("could not create window: %w", err)
	}

	val := win.Eval("loadCSS(`fonts`, `" + string(fonts) + "`);")
	if val.Err() != nil {
		return nil, fmt.Errorf("could not load fonts: %w", err)
	}
	for i, cssPath := range cfg.CustomCSS {
		b, err := os.ReadFile(filepath.Clean(cssPath))
		if err != nil {
			return nil, fmt.Errorf("could not read custom css %q: %w", cssPath, err)
		}
		name := "customCSS" + strconv.Itoa(i+1)
		val := win.Eval("loadCSS(`" + name + "`, `" + string(b) + "`);")
		if val.Err() != nil {
			return nil, fmt.Errorf("could not load custom css %q: %w", cssPath, err)
		}
	}

	return &UI{
		win: win,
	}, nil
}

// Bind binds a function into javascript.
func (ui *UI) Bind(name string, fun interface{}) error {
	return ui.win.Bind(name, fun)
}

// Eval evaluates a javascript expression.
func (ui *UI) Eval(js string) (interface{}, error) {
	v := ui.win.Eval(js)
	if v.Err() != nil {
		return nil, v.Err()
	}

	if len(v.Bytes()) == 0 {
		return nil, nil // nolint:nilnil
	}

	var i interface{}
	err := v.To(&i)
	return i, err
}

// Done returns a channel signalling the UI being closed.
func (ui *UI) Done() <-chan struct{} {
	return ui.win.Done()
}

// Close closes the ui.
func (ui *UI) Close() error {
	return ui.win.Close()
}

// UIContext implements a UI in context of a module element.
type UIContext struct {
	ui   *UI
	name string
}

// NewUIContext returns a ui with the context of a module.
func NewUIContext(ui *UI, name string, pos module.Position) (*UIContext, error) {
	name = strings.ReplaceAll(name, " ", "_")
	if _, err := ui.Eval(fmt.Sprintf(`createModule("%s", "%s", "%s");`, name, pos.Vertical, pos.Horizontal)); err != nil {
		return nil, fmt.Errorf("%s: could not create module ui element: %w", name, err)
	}

	return &UIContext{
		ui:   ui,
		name: name,
	}, nil
}

// LoadCSS loads a css style into the ui.
func (u *UIContext) LoadCSS(css string) error {
	_, err := u.ui.Eval(fmt.Sprintf("loadCSS(`%s`, `%s`);", u.name, css))
	return err
}

// LoadHTML loads html into the module.
func (u *UIContext) LoadHTML(html string) error {
	_, err := u.ui.Eval(fmt.Sprintf("loadModuleHTML(`%s`, `%s`);", u.name, html))
	return err
}

// Bind binds a function into javascript.
func (u *UIContext) Bind(name string, fun interface{}) error {
	return u.ui.Bind(name, fun)
}

// Eval evaluates a javascript expression.
func (u *UIContext) Eval(js string, ctx ...interface{}) (interface{}, error) {
	return u.ui.Eval(fmt.Sprintf(js, ctx...))
}
