package glass

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hamba/logger/v2"
	"github.com/vincent-petithory/dataurl"
	"github.com/zserge/lorca"
)

var (
	//go:embed webui/index.html
	page []byte

	//go:embed webui/wasm_exec.js
	wasmExec []byte

	//go:embed webui/fonts.css
	fonts []byte
)

// newFunc is used for testing.
var newFunc = lorca.New

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
func NewUI(cfg UIConfig, log *logger.Logger) (*UI, error) {
	// Add wasmExec to html page.
	page = bytes.Replace(page, []byte("{{ .WASMExec }}"), wasmExec, 1)

	args := []string{
		"--disable-web-security",
		"--test-type",
	}
	if cfg.Fullscreen {
		args = append(args, "--start-fullscreen")
	}
	url := dataurl.New(page, "text/html")
	win, err := newFunc(url.String(), "", cfg.Width, cfg.Height, args...)
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

	ui := &UI{
		win: win,
	}

	if err = ui.bindFuncs(log); err != nil {
		return nil, err
	}

	return ui, nil
}

// Eval evaluates a javascript expression.
func (ui *UI) Eval(js string) (any, error) {
	v := ui.win.Eval(js)
	if v.Err() != nil {
		return nil, v.Err()
	}

	if len(v.Bytes()) == 0 {
		return nil, nil //nolint:nilnil
	}

	var i any
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
