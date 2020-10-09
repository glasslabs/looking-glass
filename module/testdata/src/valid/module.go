package valid

import (
	"context"
	"io"

	"github.com/glasslabs/looking-glass/module/types"
)

type Config struct {
	Test string `yaml:"test"`
}

func NewConfig() *Config {
	return &Config{}
}

type Module struct{}

func New(ctx context.Context, cfg *Config, info types.Info, ui types.UI) (io.Closer, error) {
	return &Module{}, nil
}

func (m *Module) Close() error {
	return nil
}
