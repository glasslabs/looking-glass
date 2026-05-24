package glass

import (
	"context"
	"errors"
	"sync"

	"github.com/glasslabs/looking-glass/module"
	"github.com/glasslabs/looking-glass/ui"
	"github.com/hamba/logger/v2"
	lctx "github.com/hamba/logger/v2/ctx"
)

// uiProviderAdapter bridges ui.UI to module.UIProvider.
// ui.UI.ModuleUI returns *ui.ModuleUI (concrete), but module.UIProvider.ModuleUI
// requires module.WidgetUpdater (interface). *ui.ModuleUI satisfies the interface
// because it has Push([]byte), so the adapter coerces the return type.
type uiProviderAdapter struct {
	u *ui.UI
}

func (a uiProviderAdapter) CreateModule(name, vert, horiz string) error {
	return a.u.CreateModule(name, vert, horiz)
}

func (a uiProviderAdapter) ModuleUI(name string) module.WidgetUpdater {
	return a.u.ModuleUI(name)
}

// Run starts the looking-glass with the given configuration, and logger.
// cachePath is the filesystem path to the module cache directory.
// execCtx carries the module and assets URLs used by the module runner.
func Run(ctx context.Context, cfg Config, cachePath string, execCtx module.ExecContext, log *logger.Logger) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Debug("Creating UI window")

	gioUI := ui.New(cfg.UI, log)
	defer func() { _ = gioUI.Close() }()
	context.AfterFunc(ctx, func() { _ = gioUI.Close() })

	log.Debug("Creating module downloader", lctx.Str("cache", cachePath))

	d, err := module.NewDownloader(cachePath, log)
	if err != nil {
		return err
	}

	loader, err := module.New(ctx, uiProviderAdapter{gioUI}, d, execCtx, log)
	if err != nil {
		return err
	}
	defer func() { _ = loader.Close(context.WithoutCancel(ctx)) }()

	// Load modules in a background goroutine so the Gio window event loop
	// starts immediately. On macOS (Cocoa) the window cannot render until
	// app.Window.Event() is called; delaying that call until after all WASM
	// modules are compiled (which can take tens of seconds) means the window
	// never appears.
	var wg sync.WaitGroup
	wg.Go(func() {
		for _, desc := range cfg.Modules {
			log.Info("Loading module", lctx.Str("module", desc.Name))

			loader.Load(ctx, desc)
		}
	})
	defer wg.Wait()

	log.Debug("Starting render loop")

	if err = gioUI.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		cancel()

		return err
	}
	return nil
}
