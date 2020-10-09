package main

import (
	"os"
	"time"

	"github.com/hamba/logger"
	"github.com/urfave/cli/v2"
)

// NewLogger creates a new logger.
func NewLogger(c *cli.Context) (logger.Logger, error) {
	str := c.String(flagLogLevel)
	if str == "" {
		str = "info"
	}

	lvl, err := logger.LevelFromString(str)
	if err != nil {
		return nil, err
	}

	fmtr := newLogFormatter(c)
	h := logger.BufferedStreamHandler(os.Stdout, 2000, 1*time.Second, fmtr)
	h = logger.LevelFilterHandler(lvl, h)

	return logger.New(h), nil
}

func newLogFormatter(c *cli.Context) logger.Formatter {
	format := c.String(flagLogFormat)
	switch format {
	case "json":
		return logger.JSONFormat()

	default:
		return logger.LogfmtFormat()
	}
}
