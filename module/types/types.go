package types

// Info provides information about the module.
type Info struct {
	// Name is the instance name of the module.
	// This should be used to target a specific instance
	// of the module.
	Name string

	// Path is the full path of the module.
	// This should be used to load assets.
	Path string

	// Log is a logger instance.
	Log Logger
}

// Logger represents a logger.
type Logger interface {
	Info(msg string, ctx ...interface{})
	Error(msg string, ctx ...interface{})
}

// UI represents the ui manager.
type UI interface {
	// LoadCSS adds css for use with the module.
	LoadCSS(css string) error
	// LoadHTML loads html into the element.
	LoadHTML(html string) error
	// Bind bind a function to javascript.
	Bind(name string, fun interface{}) error
	// Eval evaluates a command in the ui.
	Eval(cmd string, ctx ...interface{}) (interface{}, error)
}
