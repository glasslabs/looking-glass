package ui

import (
	"context"
	"errors"
	"image"
	"image/color"
	"slices"
	"sync"

	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"github.com/glasslabs/client-go"
	"github.com/hamba/logger/v2"
	lctx "github.com/hamba/logger/v2/ctx"
)

const (
	gridInset    unit.Dp = 30
	moduleMargin unit.Dp = 30

	vertTop    = "top"
	vertMiddle = "middle"
	vertBottom = "bottom"

	horizLeft   = "left"
	horizCenter = "center"
	horizRight  = "right"
)

var windowBackground = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF}

// Config contains configuration for the UI window.
type Config struct {
	Width      int  `yaml:"width"`
	Height     int  `yaml:"height"`
	Fullscreen bool `yaml:"fullscreen"`
}

// Validate validates the Config.
func (c Config) Validate() error {
	if c.Width <= 0 || c.Height <= 0 {
		return errors.New("config: ui width and height must be greater than zero")
	}
	return nil
}

// UI manages the scene graph and drives the render loop.
type UI struct {
	win    *window
	shaper *text.Shaper

	mu      sync.RWMutex
	regions map[string]*region
	modules map[string]ModuleUI

	log *logger.Logger
}

// New returns a new UI.
func New(cfg Config, log *logger.Logger) *UI {
	collection := append(gofont.Collection(), loadFontFaces(log)...)
	shaper := text.NewShaper(text.NoSystemFonts(), text.WithCollection(collection))

	return &UI{
		win:     newWindow(cfg),
		shaper:  shaper,
		regions: make(map[string]*region),
		modules: make(map[string]ModuleUI),
		log:     log,
	}
}

// CreateModule registers a new module container in the given region.
func (u *UI) CreateModule(name, vert, horiz string) {
	u.mu.Lock()
	defer u.mu.Unlock()

	key := vert + ":" + horiz
	r, ok := u.regions[key]
	if !ok {
		r = &region{vert: vert, horiz: horiz}
		u.regions[key] = r
	}

	n := &moduleNode{win: u.win}
	r.addModule(n)
	u.modules[name] = n
}

// ModuleUI returns a ModuleUI scoped to the named module.
func (u *UI) ModuleUI(name string) ModuleUI {
	u.mu.RLock()
	defer u.mu.RUnlock()

	return u.modules[name]
}

// Run starts the render loop. It blocks until the window is closed or ctx is cancelled.
func (u *UI) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-u.win.Done():
			return nil
		default:
		}

		if !u.win.Frame(func(gtx layout.Context) { u.doLayout(gtx) }) {
			if u.win.exitErr != nil && !errors.Is(u.win.exitErr, context.Canceled) {
				u.log.Error("Exiting UI", lctx.Err(u.win.exitErr))
			}
			return nil
		}
	}
}

func (u *UI) doLayout(gtx layout.Context) layout.Dimensions {
	bg := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	paint.ColorOp{Color: windowBackground}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	bg.Pop()

	return layout.UniformInset(gridInset).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return u.layoutRegions(gtx)
	})
}

func (u *UI) layoutRegions(gtx layout.Context) layout.Dimensions {
	sz := gtx.Constraints.Max

	positions := [...]struct{ vert, horiz string }{
		{vertTop, horizLeft},
		{vertTop, horizRight},
		{vertTop, horizCenter},
		{vertBottom, horizLeft},
		{vertBottom, horizRight},
		{vertBottom, horizCenter},
		{vertMiddle, horizCenter},
	}

	for _, pos := range positions {
		u.mu.RLock()
		rgn := u.regions[pos.vert+":"+pos.horiz]
		u.mu.RUnlock()
		if rgn == nil {
			continue
		}

		naturalSz := rgn.size(gtx, u.shaper)
		if naturalSz.X == 0 && naturalSz.Y == 0 {
			continue
		}

		var x, y int
		switch pos.horiz {
		case horizLeft:
			x = 0
		case horizRight:
			x = sz.X - naturalSz.X
		case horizCenter:
			x = (sz.X - naturalSz.X) / 2
		}
		switch pos.vert {
		case vertTop:
			y = 0
		case vertBottom:
			y = sz.Y - naturalSz.Y
		case vertMiddle:
			y = (sz.Y - naturalSz.Y) / 2
		}

		renderGtx := gtx
		renderGtx.Constraints = layout.Constraints{
			Min: naturalSz,
			Max: naturalSz,
		}

		stack := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
		rgn.layout(renderGtx, u.shaper)
		stack.Pop()
	}

	return layout.Dimensions{Size: sz}
}

// Close closes the UI and its underlying window.
func (u *UI) Close() error {
	return u.win.Close()
}

// ModuleUI is the host-side handle for one running plugin instance.
type ModuleUI interface {
	Update(w client.Widget) error
}

type moduleNode struct {
	win  *window
	mu   sync.RWMutex
	tree node
}

// Update decodes an XML widget tree, wraps it into a node tree reusing cached
// state where content is unchanged, and stores it. Safe to call from any goroutine.
func (n *moduleNode) Update(w client.Widget) (err error) {
	n.mu.Lock()
	n.tree, err = newNode(w, n.tree)
	n.mu.Unlock()

	if err == nil && n.win != nil {
		n.win.Invalidate()
	}
	return err
}

func (n *moduleNode) size(gtx layout.Context, shaper *text.Shaper) image.Point {
	n.mu.RLock()
	tree := n.tree
	n.mu.RUnlock()

	if tree == nil {
		return image.Point{}
	}
	return tree.size(gtx, shaper)
}

func (n *moduleNode) layout(gtx layout.Context, shaper *text.Shaper) layout.Dimensions {
	n.mu.RLock()
	tree := n.tree
	n.mu.RUnlock()

	if tree == nil {
		return layout.Dimensions{}
	}
	return tree.layout(gtx, shaper)
}

// region holds the module nodes stacked vertically within one grid cell.
type region struct {
	vert  string
	horiz string

	mu      sync.RWMutex
	modules []*moduleNode
}

func (r *region) addModule(node *moduleNode) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.modules = append(r.modules, node)
}

func (r *region) size(gtx layout.Context, shaper *text.Shaper) image.Point {
	r.mu.RLock()
	mods := append([]*moduleNode(nil), r.modules...)
	r.mu.RUnlock()

	margin := gtx.Metric.Dp(moduleMargin)
	var result image.Point
	nonEmpty := 0
	for _, mod := range mods {
		s := mod.size(gtx, shaper)
		if s.X == 0 && s.Y == 0 {
			continue
		}
		if s.X > result.X {
			result.X = s.X
		}
		if nonEmpty > 0 {
			result.Y += margin
		}
		result.Y += s.Y
		nonEmpty++
	}
	return result
}

func (r *region) layout(gtx layout.Context, shaper *text.Shaper) layout.Dimensions {
	r.mu.RLock()
	mods := append([]*moduleNode(nil), r.modules...)
	r.mu.RUnlock()

	mods = slices.Clone(mods)
	mods = slices.DeleteFunc(mods, func(m *moduleNode) bool { return m.tree == nil })

	children := make([]layout.FlexChild, 0, len(mods))
	last := len(mods) - 1
	for i, mod := range mods {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			switch {
			case r.vert == vertBottom && i > 0:
				return layout.Inset{Top: moduleMargin}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return mod.layout(gtx, shaper)
				})
			case r.vert != vertBottom && i < last:
				return layout.Inset{Bottom: moduleMargin}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return mod.layout(gtx, shaper)
				})
			default:
				return mod.layout(gtx, shaper)
			}
		}))
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}
