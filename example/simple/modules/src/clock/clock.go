package clock

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/glasslabs/looking-glass/module/types"
)

// Config is the module configuration.
type Config struct {
	TimeFormat string `yaml:"timeFormat"`
	DateFormat string `yaml:"dateFormat"`
	Timezone   string `yaml:"timezone"`
}

// NewConfig creates a default configuration for the module.
func NewConfig() *Config {
	return &Config{
		TimeFormat: "15:04",
		DateFormat: "Monday, January 2",
	}
}

// Module is a clock module.
type Module struct {
	name string
	cfg  *Config
	ui   types.UI
	log  types.Logger

	loc  *time.Location
	done chan struct{}
}

// New returns a running clock module.
func New(_ context.Context, cfg *Config, info types.Info, ui types.UI) (io.Closer, error) {
	m := &Module{
		name: info.Name,
		cfg:  cfg,
		ui:   ui,
		log:  info.Log,
		done: make(chan struct{}),
	}

	if cfg.Timezone != "" {
		loc, err := time.LoadLocation(cfg.Timezone)
		if err != nil {
			return nil, fmt.Errorf("clock: invalid timezone %q: %w", cfg.Timezone, err)
		}
		m.loc = loc
	}

	css, err := ioutil.ReadFile(filepath.Join(info.Path, "assets/style.css"))
	if err != nil {
		return nil, fmt.Errorf("clock: could not read css: %w", err)
	}
	if err := ui.LoadCSS(string(css)); err != nil {
		return nil, err
	}
	html, err := ioutil.ReadFile(filepath.Join(info.Path, "assets/index.html"))
	if err != nil {
		return nil, fmt.Errorf("clock: could not read html: %w", err)
	}
	if err := ui.LoadHTML(string(html)); err != nil {
		return nil, err
	}

	go m.run()

	return m, nil
}

func (m *Module) run() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		now := time.Now()
		if m.loc != nil {
			now = now.In(m.loc)
		}
		t := now.Format(m.cfg.TimeFormat)
		d := now.Format(m.cfg.DateFormat)

		_, err := m.ui.Eval("document.querySelector('#%s .clock .time').innerHTML = '%s'", m.name, t)
		if err != nil {
			m.log.Error("could not update time", "module", "clock", "id", m.name, "error", err.Error())
		}

		_, err = m.ui.Eval("document.querySelector('#%s .clock .date').innerHTML = '%s'", m.name, d)
		if err != nil {
			m.log.Error("could not update date", "module", "clock", "id", m.name, "error", err.Error())
		}

		select {
		case <-m.done:
			return
		case <-ticker.C:
			continue
		}
	}
}

// Close stops and closes the module.
func (m *Module) Close() error {
	close(m.done)
	return nil
}
