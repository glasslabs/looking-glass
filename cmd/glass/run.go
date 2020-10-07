package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	glass "github.com/glasslabs/looking-glass"
	"github.com/glasslabs/looking-glass/module"
	"github.com/urfave/cli/v2"
)

func run(c *cli.Context) error {
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

	ui, err := glass.NewUI(cfg.UI.Width, cfg.UI.Height, cfg.UI.Fullscreen)
	if err != nil {
		return err
	}
	defer ui.Close()

	time.Sleep(time.Second)

	ctx := context.Background()
	bldr, err := module.NewBuilder(modPath)
	if err != nil {
		return err
	}
	for _, desc := range cfg.Modules {
		uiCtx, err := glass.NewUIContext(ui, desc.Name, desc.Position)
		if err != nil {
			return err
		}
		mod, err := bldr.Build(ctx, desc, uiCtx, log)
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
	cfg, err := glass.ParseConfig(in, secrets)
	if err != nil {
		return glass.Config{}, fmt.Errorf("could not parse configuration file: %w", err)
	}
	return cfg, nil
}
