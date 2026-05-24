package ui

import (
	"sync"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

type window struct {
	win *app.Window
	ops op.Ops

	done      chan struct{}
	closeOnce sync.Once
}

func newWindow(cfg Config) *window {
	w := new(app.Window)
	w.Option(
		app.Title("Glass"),
	)
	switch {
	case cfg.Fullscreen:
		w.Option(app.Fullscreen.Option())
	default:
		w.Option(app.Size(unit.Dp(cfg.Width), unit.Dp(cfg.Height)))
	}

	return &window{
		win:  w,
		done: make(chan struct{}),
	}
}

func (w *window) Done() <-chan struct{} {
	return w.done
}

func (w *window) Invalidate() {
	w.win.Invalidate()
}

// Frame blocks until a FrameEvent or DestroyEvent is received.
// On a FrameEvent it calls fn to record the scene and submits the ops.
// Returns false on DestroyEvent.
func (w *window) Frame(fn func(layout.Context)) bool {
	for {
		switch e := w.win.Event().(type) {
		case app.FrameEvent:
			gtx := app.NewContext(&w.ops, e)
			fn(gtx)
			e.Frame(gtx.Ops)
			return true
		case app.DestroyEvent:
			w.closeOnce.Do(func() { close(w.done) })
			return false
		}
	}
}

func (w *window) Close() error {
	w.win.Perform(system.ActionClose)
	return nil
}
