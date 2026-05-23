package main

import (
	"os"

	"github.com/hamba/logger/v2"
	"github.com/hamba/logger/v2/ctx"
	"github.com/urfave/cli/v3"
)

func newLogger(cmd *cli.Command) (*logger.Logger, error) {
	str := cmd.String(flagLogLevel)
	if str == "" {
		str = "info"
	}

	lvl, err := logger.LevelFromString(str)
	if err != nil {
		return nil, err
	}

	fmtr := newLogFormatter(cmd)

	tags := cmd.StringMap(flagLogCtx)

	fields := make([]logger.Field, 0, len(tags))
	for k, v := range tags {
		fields = append(fields, ctx.Str(k, v))
	}

	return logger.New(os.Stdout, fmtr, lvl).With(fields...), nil
}

func newLogFormatter(cmd *cli.Command) logger.Formatter {
	format := cmd.String(flagLogFormat)
	switch format {
	case "json":
		return logger.JSONFormat()
	case "console":
		return logger.ConsoleFormat()
	default:
		return logger.LogfmtFormat()
	}
}
