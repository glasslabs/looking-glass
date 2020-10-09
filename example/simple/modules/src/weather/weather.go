package weather

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/glasslabs/looking-glass/module/types"
)

const (
	api             = "https://api.openweathermap.org/data/2.5/"
	apiCurrentPath  = "weather"
	apiForecastPath = "forecast/daily"
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

	if err := m.loadCSS("assets/wu-icons-style.css"); err != nil {
		return nil, err
	}
	if err := m.loadCSS("assets/style.css"); err != nil {
		return nil, err
	}
	if err := m.render(data{}); err != nil {
		return nil, err
	}

	go m.run()

	return m, nil
}

func (m *Module) run() {
	c := http.Client{}

	ticker := time.NewTicker(m.cfg.Interval)
	defer ticker.Stop()

	for {
		m.log.Info("fetching weather data", "module", "weather", "id", m.name)

		data := data{}
		if err := m.request(c, apiCurrentPath, url.Values{}, &data.Current); err != nil {
			m.log.Error("could not get current weather data", "module", "weather", "id", m.name, "error", err.Error())
		}
		if err := m.request(c, apiForecastPath, url.Values{"cnt": []string{"4"}}, &data.Forecast); err != nil {
			m.log.Error("could not get current weather data", "module", "weather", "id", m.name, "error", err.Error())
		}

		if len(data.Forecast.List) > 1 {
			data.Current.Day = data.Forecast.List[0]
			data.Forecast.List = data.Forecast.List[1:]
		}
		data.Current.Icon = data.Current.Weather.Icon()
		for i := range data.Forecast.List {
			day := data.Forecast.List[i]

			t := time.Unix(day.Unix, 0)
			day.Day = t.Format("Monday")
			day.Icon = day.Weather.Icon()

			data.Forecast.List[i] = day
		}

		if err := m.render(data); err != nil {
			m.log.Error("could not render weather data", "module", "weather", "id", m.name, "error", err.Error())
		}

		select {
		case <-m.done:
			return
		case <-ticker.C:
			continue
		}
	}
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

func (m *Module) request(c http.Client, p string, qry url.Values, v interface{}) error {
	u, err := url.Parse(api + p)
	if err != nil {
		return fmt.Errorf("could not parse url: %w", err)
	}
	q := url.Values{}
	q.Set("id", m.cfg.LocationID)
	q.Set("appid", m.cfg.AppID)
	q.Set("units", m.cfg.Units)
	for k, val := range qry {
		q[k] = val
	}
	u.RawQuery = q.Encode()

	resp, err := c.Get(u.String())
	if err != nil {
		return fmt.Errorf("could not parse url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		de := dataError{}
		if err := json.NewDecoder(resp.Body).Decode(&de); err != nil {
			return fmt.Errorf("could not parse error: %w", err)
		}
		return fmt.Errorf("could not fetch data: %s", de.Message)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("could not parse data: %w", err)
	}
	return nil
}

// Close stops and closes the module.
func (m *Module) Close() error {
	close(m.done)
	return nil
}

type dataError struct {
	Code    int    `json:"cod"`
	Message string `json:"message"`
}

type data struct {
	Current  current
	Forecast forecast
}

type current struct {
	Main struct {
		Temp float64 `json:"temp"`
	} `json:"main"`
	Day     day
	Weather weather `json:"weather"`
	Icon    string
}

type forecast struct {
	List []day `json:"list"`
}

type day struct {
	Unix int64 `json:"dt"`
	Day  string
	Temp struct {
		Min float64 `json:"min"`
		Max float64 `json:"max"`
	} `json:"temp"`
	Weather weather `json:"weather"`
	Icon    string
	Rain    float64 `json:"rain"`
}

const unknownIcon = "wu-unknown"

var iconTable = map[string]string{
	"01d": "wu-clear",
	"02d": "wu-partlycloudy",
	"03d": "wu-cloudy",
	"04d": "wu-cloudy",
	"09d": "wu-flurries",
	"10d": "wu-rain",
	"11d": "wu-tstorms",
	"13d": "wu-snow",
	"50d": "wu-fog",
	"01n": "wu-clear wu-night",
	"02n": "wu-partlycloudy wu-night",
	"03n": "wu-cloudy wu-night",
	"04n": "wu-cloudy wu-night",
	"09n": "wu-flurries wu-night",
	"10n": "wu-rain wu-night",
	"11n": "wu-tstorms wu-night",
	"13n": "wu-snow wu-night",
	"50n": "wu-fog wu-night",
}

type weather []struct {
	IconCode string `json:"icon"`
}

// ResolveIcon returns the weather icon or the unknown icon.
func (w weather) Icon() string {
	if len(w) == 0 {
		return unknownIcon
	}
	icn, ok := iconTable[w[0].IconCode]
	if !ok {
		return unknownIcon
	}
	return icn
}
