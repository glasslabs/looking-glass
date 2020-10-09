package bad_return

import (
	"context"

	"github.com/glasslabs/looking-glass/module/types"
)

type Config struct {
	Test string `yaml:"test"`
}

func NewConfig() *Config {
	return &Config{}
}

func New(ctx context.Context, cfg *Config, info types.Info, ui types.UI) string {
	return "&Module{}, nil"
}
