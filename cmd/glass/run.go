package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	glass "github.com/glasslabs/looking-glass"
	"github.com/glasslabs/looking-glass/module"
	cmdx "github.com/hamba/cmd/v3"
	lctx "github.com/hamba/logger/v2/ctx"
	"github.com/hamba/pkg/v2/http/server"
	"github.com/hamba/statter/v2"
	"github.com/urfave/cli/v3"
)

func run(ctx context.Context, cmd *cli.Command) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log, err := cmdx.NewLogger(cmd)
	if err != nil {
		return err
	}

	secrets, err := loadSecrets(cmd.String(flagSecretsFile))
	if err != nil {
		return err
	}

	cfg, err := loadConfig(cmd.String(flagConfigFile), secrets)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(cmd.String(flagAssetsPath)))))
	mux.Handle("/modules/", http.StripPrefix("/modules/", http.FileServer(http.Dir(cmd.String(flagModPath)))))

	addr := cmd.String(flagAddr)
	srv := &server.GenericServer[context.Context]{
		Addr:    addr,
		Handler: mux,
		Stats:   statter.New(statter.DiscardReporter, time.Hour),
		Log:     log,
	}

	log.Info("Starting API server", lctx.Str("version", version), lctx.Str("addr", addr))
	go func() {
		if err = srv.Run(ctx); err != nil {
			log.Error("Server error", lctx.Err(err))

			cancel()
		}
	}()

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

	d, err := module.NewDownloader(cmd.String(flagModPath), log)
	if err != nil {
		return err
	}
	loader, err := module.New(ui, d, execCtx, log)
	if err != nil {
		return err
	}

	for _, desc := range cfg.Modules {
		if err = loader.Load(ctx, desc); err != nil {
			return fmt.Errorf("could not load module: %w", err)
		}
	}

	select {
	case <-ui.Done():
	case <-ctx.Done():
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
