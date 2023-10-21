package glass

import (
	"fmt"

	"github.com/hamba/logger/v2"
	lctx "github.com/hamba/logger/v2/ctx"
	"github.com/zserge/lorca"
)

func (ui *UI) bindFuncs(log *logger.Logger) error {
	return bindLogger(ui.win, log)
}

func bindLogger(ui lorca.UI, log *logger.Logger) error {
	if err := ui.Bind("log.debug", logFunc(log.Debug)); err != nil {
		return err
	}
	if err := ui.Bind("log.info", logFunc(log.Info)); err != nil {
		return err
	}
	return ui.Bind("log.error", logFunc(log.Error))
}

func logFunc(fn func(string, ...logger.Field)) func(string, []any) {
	return func(msg string, args []any) {
		if len(args)%2 == 1 {
			args = append(args, "Missing")
		}
		ctx := make([]logger.Field, 0, len(args)/2)
		for i := 0; i < len(args); i += 2 {
			k, ok := args[i].(string)
			if !ok {
				k = fmt.Sprint(args[i])
			}
			ctx = append(ctx, lctx.Interface(k, args[i+1]))
		}

		fn(msg, ctx...)
	}
}
