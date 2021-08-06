package logadpt

import (
	"github.com/hamba/logger/v2"
	logCtx "github.com/hamba/logger/v2/ctx"
)

// LogAdapter adapts Logger to the types logger.
type LogAdapter struct {
	Log *logger.Logger
}

// Info prints an informational message.
func (l LogAdapter) Info(msg string, ctx ...interface{}) {
	l.Log.Info(msg, toContext(ctx)...)
}

// Error prints an error message.
func (l LogAdapter) Error(msg string, ctx ...interface{}) {
	l.Log.Error(msg, toContext(ctx)...)
}

func toContext(ctx []interface{}) []logger.Field {
	fields := make([]logger.Field, 0, len(ctx)/2)
	for i := 0; i < len(ctx); i += 2 {
		k, ok := ctx[i].(string)
		if !ok {
			continue
		}

		fields = append(fields, logCtx.Interface(k, ctx[i+1]))
	}

	return fields
}
