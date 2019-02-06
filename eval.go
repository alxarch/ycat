package ycat

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"strings"
	"text/template"

	jsonnet "github.com/google/go-jsonnet"
)

// Eval handles Jsonnet snippet evaluation
type Eval struct {
	Bind    string
	Snippet string
	Modules map[string]string
}

// AddModule adds a module to the snippet
func (e *Eval) AddModule(name, path string) {
	if e.Modules == nil {
		e.Modules = make(map[string]string)
	}
	e.Modules[name] = path
}

var tplSnippet = template.Must(template.New("snippet").Parse(`
{{- range $name, $_ := .Modules }}
local {{$name}} = std.extVar("{{$name}}");
{{- end }}
local {{.Bind}} = std.extVar("{{.Bind}}");
{{.Snippet}}`))

// Render renders the Jsonnet snippet to be executed
func (e *Eval) Render() (string, error) {
	w := strings.Builder{}
	if err := tplSnippet.Execute(&w, e); err != nil {
		return "", err
	}
	return w.String(), nil
}

// Pipeline builds a processing pipeline step
func (e *Eval) Pipeline() (PipelineFunc, error) {
	if e.Snippet == "" {
		return nil, nil
	}
	if e.Bind == "" {
		e.Bind = "_"
	}
	snippet, err := e.Render()
	if err != nil {
		return nil, err
	}
	vm := jsonnet.MakeVM()
	//TODO: Add FileImporter
	for name, path := range e.Modules {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		vm.ExtCode(name, string(data))
	}
	return func(ctx context.Context, in <-chan *Value, out chan<- *Value) error {
		for v := range in {
			raw, err := json.Marshal(v)
			if err != nil {
				return err
			}
			vm.ExtCode(e.Bind, string(raw))
			val, err := vm.EvaluateSnippet("", snippet)
			if err != nil {
				return err
			}
			result := new(Value)
			if err := json.Unmarshal([]byte(val), result); err != nil {
				return err
			}
			select {
			case out <- result:
			case <-ctx.Done():
				return nil
			}
		}
		return nil
	}, nil
}
