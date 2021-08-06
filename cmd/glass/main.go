package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/hamba/cmd/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli/v2"
)

const (
	flagConfigFile  = "config"
	flagSecretsFile = "secrets"
	flagModPath     = "modules"
)

var version = "¯\\_(ツ)_/¯"

var commands = []*cli.Command{
	{
		Name:  "run",
		Usage: "Run looking glass",
		Flags: cmd.Flags{
			&cli.StringFlag{
				Name:    flagSecretsFile,
				Aliases: []string{"s"},
				Usage:   "The path to the secrets file.",
				EnvVars: []string{"SECRETS"},
			},
			&cli.StringFlag{
				Name:     flagConfigFile,
				Aliases:  []string{"c"},
				Usage:    "The path to the configuration file.",
				EnvVars:  []string{"CONFIG"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     flagModPath,
				Aliases:  []string{"m"},
				Usage:    "The path to the modules.",
				EnvVars:  []string{"MODULES"},
				Required: true,
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
