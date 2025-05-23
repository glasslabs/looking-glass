package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/ettle/strcase"
	"github.com/hamba/cmd/v3"
	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli/v3"
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
				Sources: cli.EnvVars(strcase.ToSNAKE(strcase.ToSNAKE(flagAddr))),
			},
			&cli.StringFlag{
				Name:    flagSecretsFile,
				Aliases: []string{"s"},
				Usage:   "The path to the secrets file.",
				Sources: cli.EnvVars(strcase.ToSNAKE(flagSecretsFile)),
			},
			&cli.StringFlag{
				Name:     flagConfigFile,
				Aliases:  []string{"c"},
				Usage:    "The path to the configuration file.",
				Required: true,
				Sources:  cli.EnvVars(strcase.ToSNAKE(flagConfigFile)),
			},
			&cli.StringFlag{
				Name:     flagAssetsPath,
				Aliases:  []string{"a"},
				Usage:    "The path to the assets.",
				Required: true,
				Sources:  cli.EnvVars(strcase.ToSNAKE(flagAssetsPath)),
			},
			&cli.StringFlag{
				Name:     flagModPath,
				Aliases:  []string{"m"},
				Usage:    "The path to the module cache.",
				Required: true,
				Sources:  cli.EnvVars(strcase.ToSNAKE(flagModPath)),
			},
		}.Merge(cmd.LogFlags),
		Action: run,
	},
}

func main() {
	os.Exit(realMain())
}

func realMain() (code int) {
	ui := newTerm()

	defer func() {
		if v := recover(); v != nil {
			ui.Error(fmt.Sprintf("Panic: %v\n%s", v, string(debug.Stack())))
			code = 1
			return
		}
	}()

	app := cli.Command{
		Name:     "looking glass",
		Usage:    "Smart mirror platform",
		Version:  version,
		Commands: commands,
		Suggest:  true,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := app.Run(ctx, os.Args); err != nil {
		ui.Error(err.Error())
		return 1
	}
	return 0
}
