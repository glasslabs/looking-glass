package glass

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
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

// Validate validates the configuration.
func (c Config) Validate() error {
	if err := c.UI.Validate(); err != nil {
		return err
	}

	if len(c.Modules) == 0 {
		return errors.New("config: at least one module is required")
	}
	seen := map[string]bool{}
	pathVer := map[string]string{}
	for _, mod := range c.Modules {
		if err := mod.Validate(); err != nil {
			return err
		}
		if seen[mod.Name] {
			return fmt.Errorf("config: module name %q is a duplicate. module names must be unique", mod.Name)
		}
		seen[mod.Name] = true

		ver, ok := pathVer[mod.Path]
		if ok && ver != mod.Version {
			return fmt.Errorf("config: module %q has mismatched versions (%s != %s)", mod.Path, mod.Version, ver)
		}
		pathVer[mod.Path] = mod.Version
	}

	return nil
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
		"Env":     getEnvVars(),
	})
	if err != nil {
		return cfg, fmt.Errorf("invalid configuration template: %w", err)
	}

	err = yaml.Unmarshal(buf.Bytes(), &cfg)
	return cfg, err
}

func getEnvVars() map[string]string {
	vars := make(map[string]string)
	for _, v := range os.Environ() {
		parts := strings.Split(v, "=")
		vars[parts[0]] = parts[1]
	}

	return vars
}
