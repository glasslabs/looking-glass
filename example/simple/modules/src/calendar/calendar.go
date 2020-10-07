package weather

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/glasslabs/looking-glass/module/types"
)

// Config is the module configuration.
type Config struct {
	LocationID string `yaml:"locationId"`
	AppID      string `yaml:"appId"`
	Units      string `yaml:"units"`

	Interval time.Duration `yaml:"interval"`
}

// NewConfig creates a default configuration for the module.
func NewConfig() *Config {
	return &Config{
		Interval: 30 * time.Minute,
	}
}

// Module is a clock module.
type Module struct {
	name string
	path string
	cfg  *Config
	ui   types.UI
	log  types.Logger

	tmpl *template.Template

	done chan struct{}
}

// New returns a running clock module.
func New(_ context.Context, cfg *Config, info types.Info, ui types.UI) (io.Closer, error) {
	html, err := ioutil.ReadFile(filepath.Join(info.Path, "assets/index.html"))
	if err != nil {
		return nil, fmt.Errorf("weather: could not read html: %w", err)
	}
	tmpl, err := template.New("html").Parse(string(html))
	if err != nil {
		return nil, fmt.Errorf("weather: could not parse html: %w", err)
	}

	m := &Module{
		name: info.Name,
		path: info.Path,
		cfg:  cfg,
		ui:   ui,
		log:  info.Log,
		tmpl: tmpl,
		done: make(chan struct{}),
	}

	if err := m.loadCSS("assets/style.css"); err != nil {
		return nil, err
	}
	if err := m.render(nil); err != nil {
		return nil, err
	}

	go m.run()

	return m, nil
}

func (m *Module) run() {
	// c := http.Client{}
	//
	// ticker := time.NewTicker(m.cfg.Interval)
	// defer ticker.Stop()
	//
	// for {
	// 	m.log.Info("fetching weather data", "module", "weather", "id", m.name)
	//
	// 	data := data{}
	// 	if err := m.request(c, apiCurrentPath, url.Values{}, &data.Current); err != nil {
	// 		m.log.Error("could not get current weather data", "module", "weather", "id", m.name, "error", err.Error())
	// 	}
	// 	if err := m.request(c, apiForecastPath, url.Values{"cnt": []string{"4"}}, &data.Forecast); err != nil {
	// 		m.log.Error("could not get current weather data", "module", "weather", "id", m.name, "error", err.Error())
	// 	}
	//
	// 	if len(data.Forecast.List) > 1 {
	// 		data.Current.Day = data.Forecast.List[0]
	// 		data.Forecast.List = data.Forecast.List[1:]
	// 	}
	// 	data.Current.Icon = data.Current.Weather.Icon()
	// 	for i := range data.Forecast.List {
	// 		day := data.Forecast.List[i]
	//
	// 		t := time.Unix(day.Unix, 0)
	// 		day.Day = t.Format("Monday")
	// 		day.Icon = day.Weather.Icon()
	//
	// 		data.Forecast.List[i] = day
	// 	}
	//
	// 	if err := m.render(data); err != nil {
	// 		m.log.Error("could not render weather data", "module", "weather", "id", m.name, "error", err.Error())
	// 	}
	//
	// 	select {
	// 	case <-m.done:
	// 		return
	// 	case <-ticker.C:
	// 		continue
	// 	}
	// }
}

func (m *Module) loadCSS(path string) error {
	css, err := ioutil.ReadFile(filepath.Join(m.path, path))
	if err != nil {
		return fmt.Errorf("weather: could not read css: %w", err)
	}
	if err := m.ui.LoadCSS(string(css)); err != nil {
		return err
	}
	return nil
}

func (m *Module) render(data interface{}) error {
	var buf bytes.Buffer
	if err := m.tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("weather: could not render html: %w", err)
	}
	return m.ui.LoadHTML(buf.String())
}

// Close stops and closes the module.
func (m *Module) Close() error {
	close(m.done)
	return nil
}
