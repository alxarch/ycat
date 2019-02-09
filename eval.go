package ycat

import (
	"encoding/json"
	"os"
	"path"
	"strings"

	jsonnet "github.com/google/go-jsonnet"
)

// Eval handles Jsonnet snippet evaluation
type Eval struct {
	Bind         string
	MaxStackSize int
	Array        bool
	Vars         map[string]Var
}

type VarType uint

const (
	_ VarType = iota
	FileVar
	CodeVar
	RawVar
)

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
			w.WriteString(`import("`)
		case ".yaml", ".yml":
			w.WriteString(`importyaml("`)
		default:
			w.WriteString(`importstr("`)
		}
		w.WriteString(v.Value)
		w.WriteString("\");\n")
	default:
		w.WriteString(`std.extVar("`)
		w.WriteString(name)
		w.WriteString("\");\n")
	}

}

func (e *Eval) Render(snippet string) string {
	w := strings.Builder{}
	for name, v := range e.Vars {
		v.Render(&w, name)
	}
	bind := bindVar(e.Bind)
	Var{Type: CodeVar}.Render(&w, bind)
	w.WriteString(snippet)
	return w.String()
}

func (e *Eval) VM(vm *jsonnet.VM) *jsonnet.VM {
	if vm == nil {
		vm = jsonnet.MakeVM()
	}
	if e.MaxStackSize > 0 {
		vm.MaxStack = e.MaxStackSize
	}

	//TODO: [eval] Add FileImporter and -J search dir args
	//TODO: [eval] Define some default helpers (ie sortBy) and bind them to _
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
	return vm

}

const DefaultInputVar = "x"

func bindVar(v string) string {
	if v == "" {
		return DefaultInputVar
	}
	return v
}

// EvalTask transforms a stream of input values with Jsonnet
func EvalTask(vm *jsonnet.VM, bind, filename, snippet string) StreamTask {
	bind = bindVar(bind)
	return StreamFunc(func(s Stream) error {
		for {
			v, ok := s.Next()
			if !ok {
				return nil
			}
			raw, err := json.Marshal(v)
			if err != nil {
				return err
			}
			vm.ExtCode(bind, string(raw))
			val, err := vm.EvaluateSnippet(filename, snippet)
			if err != nil {
				return err
			}
			result := new(Value)
			if err := json.Unmarshal([]byte(val), result); err != nil {
				return err
			}
			if !s.Push(result) {
				return nil
			}
		}
	})
}

func EvalFilename() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return path.Join(cwd, "ycat.jsonnet"), nil
}
