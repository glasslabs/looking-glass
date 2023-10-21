package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ettle/strcase"
	"github.com/hamba/cmd/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli/v2"
)

const (
	flagAddr        = "addr"
	flagConfigFile  = "config"
	flagSecretsFile = "secrets"
	flagAssetsPath  = "assets"
	flagModPath     = "modules"
)

var version = "¯\\_(ツ)_/¯"

var commands = []*cli.Command{
	{
		Name:  "run",
		Usage: "Run looking glass",
		Flags: cmd.Flags{
			&cli.StringFlag{
				Name:    flagAddr,
				Usage:   "The HTTP address to listen to.",
				Value:   "localhost:8080",
				EnvVars: []string{strcase.ToSNAKE(flagAddr)},
			},
			&cli.StringFlag{
				Name:    flagSecretsFile,
				Aliases: []string{"s"},
				Usage:   "The path to the secrets file.",
				EnvVars: []string{strcase.ToSNAKE(flagSecretsFile)},
			},
			&cli.StringFlag{
				Name:     flagConfigFile,
				Aliases:  []string{"c"},
				Usage:    "The path to the configuration file.",
				Required: true,
				EnvVars:  []string{strcase.ToSNAKE(flagConfigFile)},
			},
			&cli.StringFlag{
				Name:     flagAssetsPath,
				Aliases:  []string{"a"},
				Usage:    "The path to the assets.",
				Required: true,
				EnvVars:  []string{strcase.ToSNAKE(flagAssetsPath)},
			},
			&cli.StringFlag{
				Name:     flagModPath,
				Aliases:  []string{"m"},
				Usage:    "The path to the module cache.",
				Required: true,
				EnvVars:  []string{strcase.ToSNAKE(flagModPath)},
			},
		}.Merge(cmd.LogFlags),
		Action: run,
	},
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ui := newTerm()

	app := &cli.App{
		Name:     "looking glass",
		Usage:    "Smart mirror platform",
		Version:  version,
		Commands: commands,
	}

	if err := app.RunContext(ctx, os.Args); err != nil {
		ui.Error(err.Error())
	}
}
