package ycat

//go:generate go run gen.go

import (
	"io/ioutil"
	"os"
	"path"
	"strings"

	jsonnet "github.com/google/go-jsonnet"
)

// Eval is the execution environment for Jsonnet
type Eval struct {
	Bind         string
	MaxStackSize int
	Array        bool
	Vars         map[string]Var
	vm           *jsonnet.VM
}

// VarType is the type of an external variable
type VarType uint

// VarTypes
const (
	_ VarType = iota
	FileVar
	CodeVar
	RawVar
)

// Var is an external variable
type Var struct {
	Type  VarType
	Value string
}

// AddVar adds an external variable
func (e *Eval) AddVar(typ VarType, name, value string) {
	if e.Vars == nil {
		e.Vars = make(map[string]Var)
	}
	e.Vars[name] = Var{typ, value}
}

// Render renders the Jsonnet snippet to be executed
func (v Var) Render(w *strings.Builder, name string) {
	w.WriteString("local ")
	w.WriteString(name)
	w.WriteString(" = ")
	switch v.Type {
	case FileVar:
		switch path.Ext(v.Value) {
		case ".json", ".libsonnet", ".jsonnet":
			w.WriteString(`import "`)
		case ".yaml", ".yml":
			w.WriteString(`importyaml "`)
		default:
			w.WriteString(`importstr "`)
		}
		w.WriteString(v.Value)
		w.WriteString("\";\n")
	default:
		w.WriteString(`std.extVar("`)
		w.WriteString(name)
		w.WriteString("\");\n")
	}

}

// Render renders a snippet binding local variables
func (e *Eval) Render(snippet string) string {
	w := strings.Builder{}
	for name, v := range e.Vars {
		v.Render(&w, name)
	}
	bind := bindVar(e.Bind)
	Var{Type: CodeVar}.Render(&w, "_")
	Var{Type: CodeVar}.Render(&w, bind)
	w.WriteString(snippet)
	return w.String()
}

// VM updates or creates a Jsonnet VM
func (e *Eval) VM() (vm *jsonnet.VM) {
	if e.vm == nil {
		e.vm = jsonnet.MakeVM()
	}
	vm = e.vm
	if e.MaxStackSize > 0 {
		vm.MaxStack = e.MaxStackSize
	}

	for name, v := range e.Vars {
		switch v.Type {
		case FileVar:
			// Handled by import
		case CodeVar:
			vm.ExtCode(name, v.Value)
		default:
			vm.ExtVar(name, v.Value)
		}
	}
	vm.ExtCode("_", ycatStdLib)
	return vm

}

// DefaultInputVar is the default name for the stream value
const DefaultInputVar = "x"

func bindVar(v string) string {
	if v == "" {
		return DefaultInputVar
	}
	return v
}

func (e *Eval) SnippetFromFile(filename string) StreamTask {
	return StreamFunc(func(s Stream) error {
		snippet, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		return e.Snippet(filename, string(snippet)).Run(s)
	})
}

// EvalSnippetTask transforms a stream of input values with Jsonnet
func (e *Eval) Snippet(filename, snippet string) StreamTask {
	bind := bindVar(e.Bind)
	vm := e.VM()
	snippet = e.Render(snippet)
	return StreamFunc(func(s Stream) error {
		for {
			v, ok := s.Next()
			if !ok {
				return nil
			}
			vm.ExtCode(bind, v.MarshalJSONString())
			result, err := vm.EvaluateSnippet(filename, snippet)
			if err != nil {
				return err
			}
			if !s.Push(RawValue(result)) {
				return nil
			}
		}
	})
}

// EvalFilename returns a filename on CWD
func EvalFilename() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return path.Join(cwd, "ycat.jsonnet"), nil
}
