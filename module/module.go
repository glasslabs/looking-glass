package module

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	stypes "github.com/glasslabs/looking-glass/module/internal/types"
	"github.com/glasslabs/looking-glass/module/types"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"github.com/traefik/yaegi/stdlib/unsafe"
	"golang.org/x/mod/module"
	"gopkg.in/yaml.v3"
)

const (
	srcPath    = "src"
	markerFile = ".looking-glass"
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

var modNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)

// Descriptor describes the module and its configuration.
type Descriptor struct {
	Name     string    `yaml:"name"`
	Path     string    `yaml:"path"`
	Version  string    `yaml:"version"`
	Package  string    `yaml:"package"`
	Position Position  `yaml:"position"`
	Config   yaml.Node `yaml:"config"`
}

// Validate validates a module descriptor.
func (d Descriptor) Validate() error {
	if d.Name == "" {
		return errors.New("config: a module must have a name")
	}
	if !modNameRegex.Match([]byte(d.Name)) {
		return fmt.Errorf("%s: module names may only contain letters, numbers, '-' and '_'", d.Name)
	}

	if d.Path == "" {
		return fmt.Errorf("%s: module must have a path", d.Name)
	}

	return nil
}

// Service extracts and runs modules.
type Service struct {
	path string
	c    Client

	Debug func(msg string, ctx ...interface{})
}

// NewService returns a module service.
func NewService(modPath string, c Client) (Service, error) {
	p, err := filepath.Abs(modPath)
	if err != nil {
		return Service{}, fmt.Errorf("invalid module path %q: %w", modPath, err)
	}

	return Service{
		path: p,
		c:    c,
	}, nil
}

func (s Service) debug(msg string, ctx ...interface{}) {
	if s.Debug == nil {
		return
	}
	s.Debug(msg, ctx...)
}

// Extract downloads and extracts a module into the module path.
func (s Service) Extract(desc Descriptor) error {
	if desc.Version == "" {
		// User is not expecting us to extract. Nothing to do.
		return nil
	}
	m, err := s.c.Version(desc.Path, desc.Version)
	if err != nil {
		return err
	}
	s.debug("module version resolved", "module", m.Path, "ver", m.Version)

	modPath := filepath.Join(s.path, srcPath, desc.Path)
	markerPath := filepath.Join(modPath, markerFile)
	if _, err := os.Stat(modPath); err == nil {
		// This might be a user controlled path, check for the marker.
		if _, err := os.Stat(markerPath); err != nil {
			s.debug("path seems to be a user module path", "path", modPath)
			// Not our path or something we cannot touch.
			return nil
		}
		if ver, err := ioutil.ReadFile(markerPath); err == nil && m.Version == string(ver) {
			s.debug("module is at correct version", "path", modPath)
			// The correct version is already extracted. Nothing to do.
			return nil
		}

		// The path exists but is the wrong version, remove it.
		s.debug("cleaning module path", "path", modPath)
		if err := s.cleanPath(modPath); err != nil {
			return fmt.Errorf("could not remove old module: %w", err)
		}
	}

	s.debug("extracting module", "path", m.Path, "ver", m.Version)
	z, err := s.c.Download(m)
	if err != nil {
		return err
	}
	defer z.Close()
	if err := os.MkdirAll(modPath, 0777); err != nil {
		return fmt.Errorf("could not create module path %q: %w", modPath, err)
	}
	if err := s.unzip(z, m, modPath); err != nil {
		return fmt.Errorf("could not extract module: %w", err)
	}
	if err := ioutil.WriteFile(markerPath, []byte(m.Version), 0444); err != nil {
		return fmt.Errorf("could not write module marker: %w", err)
	}

	return nil
}

func (s Service) cleanPath(path string) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(path, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s Service) unzip(r io.Reader, m module.Version, path string) error {
	var buf bytes.Buffer
	size, err := io.Copy(&buf, r)
	if err != nil {
		return err
	}
	br := bytes.NewReader(buf.Bytes())

	z, err := zip.NewReader(br, size)
	if err != nil {
		return err
	}
	prefix := fmt.Sprintf("%s@%s/", m.Path, m.Version)
	for _, zf := range z.File {
		if !strings.HasPrefix(zf.Name, prefix) {
			return fmt.Errorf("unexpected file name %s", zf.Name)
		}
		name := zf.Name[len(prefix):]
		if name == "" || strings.HasSuffix(name, "/") {
			continue
		}
		dst := filepath.Join(path, name)
		if err := os.MkdirAll(filepath.Dir(dst), 0777); err != nil {
			return err
		}
		w, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0444)
		if err != nil {
			return err
		}
		r, err := zf.Open()
		if err != nil {
			w.Close()
			return err
		}
		lr := &io.LimitedReader{R: r, N: int64(zf.UncompressedSize64) + 1}
		_, err = io.Copy(w, lr)
		r.Close()
		if err != nil {
			w.Close()
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		if lr.N <= 0 {
			return fmt.Errorf("file %s is larger than declared (%d bytes)", zf.Name, zf.UncompressedSize64)
		}
	}

	return nil
}

var (
	errorType  = reflect.TypeOf((*error)(nil)).Elem()
	closerType = reflect.TypeOf((*io.Closer)(nil)).Elem()
)

// Run builds and runs a module.
func (s Service) Run(ctx context.Context, desc Descriptor, ui types.UI, log types.Logger) (io.Closer, error) {
	pkg := desc.Package
	if pkg == "" {
		base := path.Base(desc.Path)
		pkg = base[strings.LastIndex(base, "-")+1:]
	}

	i := interp.New(interp.Options{GoPath: s.path})
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
		Path: filepath.Join(s.path, srcPath, desc.Path),
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
