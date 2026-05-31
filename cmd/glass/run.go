package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	glass "github.com/glasslabs/looking-glass"
	"github.com/glasslabs/looking-glass/module"
	lctx "github.com/hamba/logger/v2/ctx"
	"github.com/urfave/cli/v3"
)

func run(ctx context.Context, cmd *cli.Command) error {
	log, err := newLogger(cmd)
	if err != nil {
		return err
	}

	log.Info("Starting Looking Glass", lctx.Str("version", version))

	secrets, err := loadSecrets(cmd.String(flagSecretsFile))
	if err != nil {
		return err
	}

	cfg, err := loadConfig(cmd.String(flagConfigFile), secrets)
	if err != nil {
		return err
	}

	execCtx := module.ExecContext{
		AssetsPath: cmd.String(flagAssetsPath),
	}
	if err = glass.Run(ctx, cfg, cmd.String(flagModPath), execCtx, log); err != nil {
		log.Error("Looking Glass Shutdown", lctx.Err(err))

		return err
	}

	log.Info("Looking Glass Shutdown")

	return nil
}

func loadSecrets(file string) (map[string]any, error) {
	if file == "" {
		return nil, nil //nolint:nilnil
	}

	in, err := os.ReadFile(filepath.Clean(file))
	if errors.Is(err, os.ErrNotExist) {
		//nolint:nilnil
		return nil, nil
	}
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
