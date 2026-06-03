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
	win        *app.Window
	fullscreen bool
	ops        op.Ops

	done      chan struct{}
	closeOnce sync.Once

	exitErr error
}

func newWindow(cfg Config) *window {
	opts := []app.Option{
		app.Title("Glass"),
		app.MinSize(unit.Dp(640), unit.Dp(480)),
		app.Size(unit.Dp(cfg.Width), unit.Dp(cfg.Height)),
	}

	w := &app.Window{}
	w.Option(opts...)

	return &window{
		win:        w,
		fullscreen: cfg.Fullscreen,
		done:       make(chan struct{}),
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

			// If fullscreen is requested, set it now, so we don't panic
			// in Wayland.
			if w.fullscreen {
				w.fullscreen = false
				w.win.Option(app.Fullscreen.Option())
			}
			return true
		case app.DestroyEvent:
			w.exitErr = e.Err
			w.closeOnce.Do(func() { close(w.done) })
			return false
		}
	}
}

func (w *window) Close() error {
	w.win.Perform(system.ActionClose)
	return nil
}
