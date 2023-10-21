package module

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
)

// GoRunner runs Go WASM modules.
type GoRunner struct {
	ui  UI
	env string

	tmpl *template.Template
}

// NewGoRunner returns a Go WASM module runner.
func NewGoRunner(ui UI, env map[string]string) (GoRunner, error) {
	tmpl, err := template.New("module").Parse(goModuleScriptTmpl)
	if err != nil {
		return GoRunner{}, fmt.Errorf("parsing template: %w", err)
	}

	b, err := json.Marshal(env)
	if err != nil {
		return GoRunner{}, fmt.Errorf("encoding: %w", err)
	}

	return GoRunner{
		ui:   ui,
		env:  string(b),
		tmpl: tmpl,
	}, nil
}

// Run runs the module.
func (r GoRunner) Run(name, url string, cfg map[string]any) error {
	cfgStr, err := r.encodeConfig(cfg)
	if err != nil {
		return fmt.Errorf("%s: reading config: %w", name, err)
	}

	var buf bytes.Buffer
	err = r.tmpl.Execute(&buf, goTmplData{
		Name:   name,
		URL:    url,
		Config: cfgStr,
		Env:    r.env,
	})
	if err != nil {
		return fmt.Errorf("%s: creating script: %w", name, err)
	}

	if _, err = r.ui.Eval(fmt.Sprintf(`loadModule(%q);`, buf.String())); err != nil {
		return fmt.Errorf("%s: loading module: %w", name, err)
	}
	return nil
}

func (r GoRunner) encodeConfig(v map[string]any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("encoding: %w", err)
	}
	return string(b), nil
}

type goTmplData struct {
	Name   string
	URL    string
	Config string
	Env    string
}

const goModuleScriptTmpl = `
(function () {
	let go = new Go();
	WebAssembly.instantiateStreaming(fetch('{{ .URL }}'), go.importObject).then((result) => {
		go.argv = ['{{ .Name }}', '{{ .Config }}'];
		go.env = {{ .Env }};
		go.run(result.instance);
	}).catch((err) => {
		console.log(err);
	});
})();
`
