package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"gioui.org/app"
	"github.com/ettle/strcase"
	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli/v3"
)

const (
	flagConfigFile  = "config"
	flagSecretsFile = "secrets"
	flagAssetsPath  = "assets"
	flagModPath     = "modules"
	flagLogFormat   = "log.format"
	flagLogLevel    = "log.level"
	flagLogCtx      = "log.ctx"
)

const categoryLog = "Logging"

var version = "¯\\_(ツ)_/¯"

var commands = []*cli.Command{
	{
		Name:  "run",
		Usage: "Run looking glass",
		Flags: []cli.Flag{
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
				Usage:    "The path to the assets directory.",
				Required: true,
				Sources:  cli.EnvVars(strcase.ToSNAKE(flagAssetsPath)),
			},
			&cli.StringFlag{
				Name:     flagModPath,
				Aliases:  []string{"m"},
				Usage:    "The path to the module cache directory.",
				Required: true,
				Sources:  cli.EnvVars(strcase.ToSNAKE(flagModPath)),
			},
			&cli.StringFlag{
				Name:     flagLogFormat,
				Category: categoryLog,
				Usage:    "Specify the format of logs. Supported formats: 'logfmt', 'json', 'console'",
				Sources:  cli.EnvVars(strcase.ToSNAKE(flagLogFormat)),
			},
			&cli.StringFlag{
				Name:     flagLogLevel,
				Category: categoryLog,
				Value:    "info",
				Usage:    "Specify the log level. e.g. 'debug', 'info', 'error'.",
				Sources:  cli.EnvVars(strcase.ToSNAKE(flagLogLevel)),
			},
			&cli.StringMapFlag{
				Name:     flagLogCtx,
				Category: categoryLog,
				Usage:    "A list of context field appended to every log. Format: key=value.",
				Sources:  cli.EnvVars(strcase.ToSNAKE(flagLogCtx)),
			},
		},
		Action: run,
	},
}

func main() {
	// app.Main must run on the OS main thread for Gio windowing to work on
	// macOS (Cocoa) and some other platforms. The application logic runs in a
	// goroutine and signals completion via the exit code channel.
	go func() {
		code := realMain()
		os.Exit(code)
	}()
	app.Main()
}

func realMain() (code int) {
	defer func() {
		if v := recover(); v != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Panic: %v\n%s", v, string(debug.Stack()))
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
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
		return 1
	}
	return 0
}
