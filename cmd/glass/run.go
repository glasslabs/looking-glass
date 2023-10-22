package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	glass "github.com/glasslabs/looking-glass"
	"github.com/glasslabs/looking-glass/module"
	"github.com/hamba/cmd/v2"
	lctx "github.com/hamba/logger/v2/ctx"
	httpx "github.com/hamba/pkg/v2/http"
	"github.com/urfave/cli/v2"
)

func run(c *cli.Context) error {
	ctx, cancel := context.WithCancel(c.Context)
	defer cancel()

	log, err := cmd.NewLogger(c)
	if err != nil {
		return err
	}
	logCancel := log.WithTimestamp()
	defer logCancel()

	secrets, err := loadSecrets(c.String(flagSecretsFile))
	if err != nil {
		return err
	}

	cfg, err := loadConfig(c.String(flagConfigFile), secrets)
	if err != nil {
		return err
	}

	addr := c.String(flagAddr)
	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(c.String(flagAssetsPath)))))
	mux.Handle("/modules/", http.StripPrefix("/modules/", http.FileServer(http.Dir(c.String(flagModPath)))))

	log.Info("Starting API server",
		lctx.Str("ver", version),
		lctx.Str("addr", addr),
	)

	srv := httpx.NewServer(ctx, addr, mux, httpx.WithH2C())
	srv.Serve(func(err error) {
		log.Error("Server error", lctx.Err(err))
		cancel()
	})
	defer func() { _ = srv.Close() }()

	ui, err := glass.NewUI(cfg.UI, log)
	if err != nil {
		return err
	}
	defer func() {
		_ = ui.Close()
	}()

	execCtx := module.ExecContext{
		ModuleURL: "http://" + addr + "/modules",
		AssetsURL: "http://" + addr + "/assets",
	}

	d, err := module.NewDownloader(c.String(flagModPath), log)
	if err != nil {
		return err
	}
	loader, err := module.New(ui, d, execCtx, log)
	if err != nil {
		return err
	}

	for _, desc := range cfg.Modules {
		desc := desc

		if err = loader.Load(ctx, desc); err != nil {
			return fmt.Errorf("could not load module: %w", err)
		}
	}

	select {
	case <-ui.Done():
	case <-c.Context.Done():
	}

	cancel()

	return nil
}

func loadSecrets(file string) (map[string]any, error) {
	if file == "" {
		return nil, nil //nolint:nilnil
	}

	in, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, fmt.Errorf("could not read secrets file: %w", err)
	}
	s, err := glass.ParseSecrets(in)
	if err != nil {
		return nil, fmt.Errorf("could not parse secrets file: %w", err)
	}
	return s, nil
}

func loadConfig(file string, secrets map[string]any) (glass.Config, error) {
	in, err := os.ReadFile(filepath.Clean(file))
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
