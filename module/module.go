package module

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	stypes "github.com/glasslabs/looking-glass/module/internal/types"
	"github.com/glasslabs/looking-glass/module/types"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"github.com/traefik/yaegi/stdlib/unsafe"
	"gopkg.in/yaml.v3"
)

// Module positions.
const (
	Top    = "top"
	Middle = "middle"
	Bottom = "bottom"

	Left   = "left"
	Center = "center"
	Right  = "right"
)

// Position is a module position in the grid.
type Position struct {
	Vertical   string
	Horizontal string
}

// UnmarshalYAML unmarshals a Position from YAML.
func (p *Position) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var pos string
	if err := unmarshal(&pos); err != nil {
		return err
	}

	parts := strings.Split(pos, ":")
	if len(parts) != 2 {
		return errors.New("invalid position: " + pos)
	}

	switch parts[0] {
	case Top:
		p.Vertical = Top
	case Middle:
		p.Vertical = Middle
	case Bottom:
		p.Vertical = Bottom
	default:
		return errors.New("invalid vertical position: " + parts[0])
	}

	switch parts[1] {
	case Left:
		p.Horizontal = Left
	case Center:
		p.Horizontal = Center
	case Right:
		p.Horizontal = Right
	default:
		return errors.New("invalid horizontal position: " + parts[1])
	}
	return nil
}

// Descriptor describes the module and its configuration.
type Descriptor struct {
	Name     string    `yaml:"name"`
	Path     string    `yaml:"path"`
	Package  string    `yaml:"package"`
	Position Position  `yaml:"position"`
	Config   yaml.Node `yaml:"config"`
}

// TODO(nick): Validate the Descriptors Config

// Builder builds modules.
type Builder struct {
	path string
}

// NewBuilder returns a module builder.
func NewBuilder(path string) (*Builder, error) {
	p, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid module path: %w", err)
	}

	return &Builder{
		path: p,
	}, nil
}

var (
	errorType  = reflect.TypeOf((*error)(nil)).Elem()
	closerType = reflect.TypeOf((*io.Closer)(nil)).Elem()
)

// Build builds the module described by the descriptor.
func (b *Builder) Build(ctx context.Context, desc Descriptor, ui types.UI, log types.Logger) (io.Closer, error) {
	pkg := desc.Package
	if pkg == "" {
		base := path.Base(desc.Path)
		pkg = base[strings.LastIndex(base, "-")+1:]
	}

	i := interp.New(interp.Options{GoPath: b.path})
	i.Use(stdlib.Symbols)
	i.Use(unsafe.Symbols)
	i.Use(stypes.Symbols)

	_, err := i.Eval(fmt.Sprintf(`import "%s"`, desc.Path))
	if err != nil {
		return nil, fmt.Errorf("%s: could not import module %q: %w", desc.Name, desc.Path, err)
	}

	vCfg, err := i.Eval(pkg + ".NewConfig()")
	if err != nil {
		return nil, fmt.Errorf("module: could not run NewConfig: %w", err)
	}
	if err := desc.Config.Decode(vCfg.Interface()); err != nil {
		return nil, fmt.Errorf("%s: could not decode configuration: %w", desc.Name, err)
	}

	vNew, err := i.Eval(pkg + ".New")
	if err != nil {
		return nil, fmt.Errorf("module: could not find New: %w", err)
	}
	if !vNew.IsValid() || vNew.Kind() != reflect.Func ||
		vNew.Type().NumOut() != 2 ||
		vNew.Type().Out(0) != closerType || vNew.Type().Out(1) != errorType {
		return nil, fmt.Errorf("%s: module New must be a func with return '(io.Closer, error)'", desc.Name)
	}
	info := types.Info{
		Name: desc.Name,
		Path: filepath.Join(b.path, "src", desc.Path),
		Log:  log,
	}
	args := []reflect.Value{reflect.ValueOf(ctx), vCfg, reflect.ValueOf(info), reflect.ValueOf(ui)}
	res := vNew.Call(args)
	vMod, vErr := res[0], res[1]
	if vErr.Interface() != nil {
		return nil, fmt.Errorf("%s: error loading module: %w", desc.Name, vErr.Interface().(error))
	}
	if vMod.Interface() == nil {
		return nil, fmt.Errorf("%s: nil module returned", desc.Name)
	}
	return vMod.Interface().(io.Closer), nil
}
