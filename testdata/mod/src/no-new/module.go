package new_func

type Config struct {
	Test string `yaml:"test"`
}

func NewConfig() *Config {
	return &Config{}
}

type Module struct{}

func (m *Module) Close() error {
	return nil
}
