package main

import (
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli/v2"
)

const (
	flagConfigFile  = "config"
	flagSecretsFile = "secrets"
	flagModPath     = "modules"

	flagLogFormat = "log.format"
	flagLogLevel  = "log.level"
)

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

			&cli.StringFlag{
				Name:    flagLogFormat,
				Usage:   "Specify the format of logs. Supported formats: 'logfmt', 'json'",
				EnvVars: []string{"LOG_FORMAT"},
			},
			&cli.StringFlag{
				Name:    flagLogLevel,
				Value:   "info",
				Usage:   "Specify the log level. E.g. 'debug', 'warn'.",
				EnvVars: []string{"LOG_LEVEL"},
			},
		},
		Action: run,
	},
}

func main() {
	app := &cli.App{
		Name:     "looking glass",
		Usage:    "Smart mirror platform",
		Version:  version,
		Commands: commands,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
