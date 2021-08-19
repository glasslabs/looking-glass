package main

import (
	"fmt"
	"os"
	"path/filepath"

	glass "github.com/glasslabs/looking-glass"
	"github.com/glasslabs/looking-glass/internal/logadpt"
	"github.com/glasslabs/looking-glass/module"
	"github.com/hamba/cmd/v2"
	"github.com/urfave/cli/v2"
)

const proxyURL = "https://proxy.golang.org"

func run(c *cli.Context) error {
	log, err := cmd.NewLogger(c)
	if err != nil {
		return err
	}

	secrets, err := loadSecrets(c.String(flagSecretsFile))
	if err != nil {
		return err
	}

	cfg, err := loadConfig(c.String(flagConfigFile), secrets)
	if err != nil {
		return err
	}

	ui, err := glass.NewUI(cfg.UI)
	if err != nil {
		return err
	}
	defer func() {
		_ = ui.Close()
	}()

	modPath := c.String(flagModPath)
	cachePath, err := ensureCachePath(modPath)
	if err != nil {
		return err
	}
	client, err := newModuleClient(proxyURL, cachePath)
	if err != nil {
		return err
	}
	svc, err := module.NewService(modPath, client)
	if err != nil {
		return err
	}
	svc.Debug = log.Debug
	for _, desc := range cfg.Modules {
		if err = svc.Extract(desc); err != nil {
			return err
		}

		uiCtx, err := glass.NewUIContext(ui, desc.Name, desc.Position)
		if err != nil {
			return err
		}
		mod, err := svc.Run(c.Context, desc, uiCtx, logadpt.LogAdapter{Log: log})
		if err != nil {
			return err
		}
		defer func() {
			_ = mod.Close()
		}()
	}

	select {
	case <-ui.Done():
	case <-c.Context.Done():
	}

	return nil
}

func loadSecrets(file string) (map[string]interface{}, error) {
	if file == "" {
		return nil, nil
	}

	in, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read secrets file: %w", err)
	}
	s, err := glass.ParseSecrets(in)
	if err != nil {
		return nil, fmt.Errorf("could not parse secrets file: %w", err)
	}
	return s, nil
}

func loadConfig(file string, secrets map[string]interface{}) (glass.Config, error) {
	in, err := os.ReadFile(file)
	if err != nil {
		return glass.Config{}, fmt.Errorf("could not read configuration file: %w", err)
	}
	cfg, err := glass.ParseConfig(in, filepath.Dir(file), secrets)
	if err != nil {
		return glass.Config{}, fmt.Errorf("could not parse configuration file: %w", err)
	}
	if err = cfg.Validate(); err != nil {
		return glass.Config{}, err
	}
	return cfg, nil
}

func ensureCachePath(modPath string) (string, error) {
	p := filepath.Join(modPath, "cache")
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	if err := os.MkdirAll(p, 0o750); err != nil {
		return "", fmt.Errorf("could not create cache path %q: %w", p, err)
	}
	return p, nil
}

func newModuleClient(proxyURL, cachePath string) (module.Client, error) {
	pc, err := module.NewProxyClient(proxyURL)
	if err != nil {
		return nil, err
	}
	return module.NewCachedClient(pc, cachePath)
}
