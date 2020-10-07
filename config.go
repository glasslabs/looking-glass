package glass

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/glasslabs/looking-glass/module"
	"gopkg.in/yaml.v3"
)

// ParseSecrets parses secrets from in.
func ParseSecrets(in []byte) (map[string]interface{}, error) {
	sec := map[string]interface{}{}
	err := yaml.Unmarshal(in, &sec)
	return sec, err
}

// Config contains the main configuration.
type Config struct {
	UI      UIConfig            `yaml:"ui"`
	Modules []module.Descriptor `yaml:"modules"`
}

// UIConfig contains configuration for the UI.
type UIConfig struct {
	Width      int  `yaml:"width"`
	Height     int  `yaml:"height"`
	Fullscreen bool `yaml:"fullscreen"`
}

func defaultConfig() Config {
	return Config{
		UI: UIConfig{
			Width:      640,
			Height:     480,
			Fullscreen: true,
		},
	}
}

// ParseConfig parses configuration from in.
func ParseConfig(in []byte, secrets map[string]interface{}) (Config, error) {
	cfg := defaultConfig()

	tmpl, err := template.New("config").
		Parse(string(in))
	if err != nil {
		return cfg, fmt.Errorf("invalid configuration template: %w", err)
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]interface{}{
		"Secrets": secrets,
	})
	if err != nil {
		return cfg, fmt.Errorf("invalid configuration template: %w", err)
	}

	err = yaml.Unmarshal(buf.Bytes(), &cfg)
	return cfg, err
}
