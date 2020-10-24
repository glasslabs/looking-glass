package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	glass "github.com/glasslabs/looking-glass"
	"github.com/glasslabs/looking-glass/module"
	"github.com/urfave/cli/v2"
)

const proxyURL = "https://proxy.golang.org"

func run(c *cli.Context) error {
	ctx := context.Background()
	log, err := NewLogger(c)
	if err != nil {
		return err
	}

	secretFile := c.String(flagSecretsFile)
	cfgPath := c.String(flagConfigFile)
	modPath := c.String(flagModPath)

	secrets, err := loadSecrets(secretFile)
	if err != nil {
		return err
	}

	cfg, err := loadConfig(cfgPath, secrets)
	if err != nil {
		return err
	}

	ui, err := glass.NewUI(cfg.UI)
	if err != nil {
		return err
	}
	defer ui.Close()

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
		if err := svc.Extract(desc); err != nil {
			return err
		}

		uiCtx, err := glass.NewUIContext(ui, desc.Name, desc.Position)
		if err != nil {
			return err
		}
		mod, err := svc.Run(ctx, desc, uiCtx, log)
		if err != nil {
			return err
		}
		defer mod.Close()
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ui.Done():
	case <-sigs:
	}

	return nil
}

func loadSecrets(file string) (map[string]interface{}, error) {
	if file == "" {
		return nil, nil
	}

	in, err := ioutil.ReadFile(file)
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
	in, err := ioutil.ReadFile(file)
	if err != nil {
		return glass.Config{}, fmt.Errorf("could not read configuration file: %w", err)
	}
	cfg, err := glass.ParseConfig(in, filepath.Dir(file), secrets)
	if err != nil {
		return glass.Config{}, fmt.Errorf("could not parse configuration file: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return glass.Config{}, err
	}
	return cfg, nil
}

func ensureCachePath(modPath string) (string, error) {
	p := filepath.Join(modPath, "cache")
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	if err := os.MkdirAll(p, 0777); err != nil {
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
