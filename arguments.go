package ycat

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	jsonnet "github.com/google/go-jsonnet"
)

// Usage for ycat cmd
const Usage = `
ycat - command line YAML/JSON processor

USAGE:
    ycat [OPTIONS] [INPUT...]
    ycat [OPTIONS] [PIPELINE...]

OPTIONS:
    -o, --out {json|j|yaml|y}    Set output format
    -h, --help                   Show help and exit

INPUT:
    [FILE...]                    Read values from file(s)
    -y, --yaml [FILE...]         Read YAML values from file(s)
    -j, --json [FILE...]         Read JSON values from file(s)
    -n, --null                   Inject a null value 
    -a, --array                  Merge values to array

PIPELINE:
    [INPUT...] [ENV...] EVAL

ENV:
    -v, --var <VAR>=<CODE>       Bind Jsonnet variable to code
              <VAR>==<VALUE>     Bind Jsonnet variable to a string value
    -i, --import <VAR>=<FILE>    Import file into a local Jsonnet variable
        --input-var <VAR>        Change the name of the input value variable (default x) 
        --max-stack <SIZE>       Jsonnet VM max stack size (default 500)

EVAL:
    -e, --eval <SNIPPET>         Process values with Jsonnet


If no INPUT is specified, values are read from stdin as YAML.
If FILE is "-" or "" values are read from stdin until EOF.
If FILE has no type option format is detected from extension.
`

func splitArgV(s string) (string, string) {
	if n := strings.IndexByte(s, '='); 0 <= n && n < len(s) {
		return s[:n], s[n+1:]
	}
	return s, ""
}

func peekArg(args []string) (string, bool) {
	if len(args) > 0 {
		return args[0], true
	}
	return "", false
}

func (p *argParser) Eval() *Eval {
	if p.eval == nil {
		p.eval = new(Eval)
	}
	return p.eval

}
func (p *argParser) parseLong(name, value string, argv []string) ([]string, error) {
	switch name {
	case "max-stack":
		value, argv = shiftArgV(value, argv)
		size, err := strconv.Atoi(value)
		if err != nil {
			return argv, fmt.Errorf("Invalid max stack size: %s", err)
		}
		p.Eval().MaxStackSize = size
	case "input-var":
		value, argv = shiftArgV(value, argv)
		p.Eval().Bind = value
	case "import":
		value, argv = shiftArgV(value, argv)
		name, file := splitArgV(value)
		if file == "" {
			return argv, errors.New("Missing import file")
		}
		p.Eval().AddVar(FileVar, name, file)
	case "var":
		value, argv = shiftArgV(value, argv)
		name, value := splitArgV(value)
		typ := RawVar
		if len(value) > 0 && value[0] == '=' {
			value = value[1:]
			typ = CodeVar
		}
		p.Eval().AddVar(typ, name, value)
	case "eval":
		e := p.Eval()
		e.Snippet = value
		snippet := e.Render(e.Bind)
		p.vm = e.VM(p.vm)
		filename, err := EvalFilename()
		if err != nil {
			return argv, err
		}
		eval := EvalTask(p.vm, e.Bind, filename, snippet)
		if input := p.inputTask(); len(input) == 0 {
			p.tasks = append(p.tasks, eval)
		} else {
			p.tasks = append(p.tasks, input, eval)
		}
	case "output":
		value, argv = shiftArgV(value, argv)
		if p.output = OutputFromString(value); p.output == OutputInvalid {
			return argv, fmt.Errorf("Invalid output format: %q", value)
		}
	case "null":
		p.input = append(p.input[:0], NullStream{})
	case "to-json":
		p.output = OutputJSON
	case "help":
		p.help = true
	case "array":
		p.tasks = append(p.tasks, ToArray{})
	case "yaml":
		return p.parseFiles(value, argv, YAML), nil
	case "json":
		return p.parseFiles(value, argv, JSON), nil
	default:
		return argv, fmt.Errorf("Invalid option: %q", name)
	}
	return argv, nil
}

// func (p *argParser) lastTask() StreamTask {
// 	if len(p.tasks) > 0 {
// 		return p.tasks[len(p.tasks)-1]
// 	}
// 	return nil
// }

func (p *argParser) parseShort(a string, argv []string) ([]string, error) {
	for ; len(a) > 0; a = a[1:] {
		switch c := a[0]; c {
		case 'j':
			return p.parseFiles(a[1:], argv, JSON), nil
		case 'y':
			return p.parseFiles(a[1:], argv, YAML), nil
		case 'i':
			return p.parseLong("import", a[1:], argv)
		case 'v':
			return p.parseLong("var", a[1:], argv)
		case 'e':
			return p.parseLong("eval", a[1:], argv)
		case 'o':
			return p.parseLong("output", a[1:], argv)
		case 'n':
			return p.parseLong("null", a[1:], argv)
		case 'a':
			return p.parseLong("array", a[1:], argv)
		case 'h':
			p.help = true
		default:
			return argv, fmt.Errorf("Invalid short option: -%c", c)
		}
	}
	return argv, nil
}

func shiftArgV(v string, argv []string) (string, []string) {
	if len(v) > 0 {
		if v[0] == '=' {
			v = v[1:]
		}
	} else if len(argv) > 0 && !isOption(argv[0]) {
		v = argv[0]
		argv = argv[1:]
	}
	return v, argv
}
func (p *argParser) parseFiles(path string, argv []string, format Format) []string {
	switch {
	case len(path) > 0:
		if path[0] == '=' {
			path = path[1:]
		}
		p.addFile(path, format)
	case len(argv) == 0:
		p.addFile("", format)
	default:
		for ; len(argv) > 0 && !isOption(argv[0]); argv = argv[1:] {
			p.addFile(argv[0], format)
		}
	}
	return argv
}

func isOption(a string) bool {
	return len(a) > 1 && a[0] == '-' && (a[1] != '-' || len(a) > 2)
}

type argParser struct {
	vm     *jsonnet.VM
	eval   *Eval
	output Output
	input  []StreamTask
	tasks  []StreamTask
	help   bool
	err    error
}

func (p *argParser) addFile(path string, format Format) {
	f := InputFile{format, path}
	p.input = append(p.input, &f)
}

func ParseArgs(argv []string) ([]StreamTask, bool, error) {
	p := argParser{}
	if err := p.Parse(argv); err != nil {
		return nil, false, err
	}
	return p.Tasks(), p.help, nil
}

func (p *argParser) Parse(argv []string) (err error) {
	for err == nil && len(argv) > 0 {
		a := argv[0]
		argv = argv[1:]
		if len(a) > 1 && a[0] == '-' {
			switch c := a[1]; c {
			case '-':
				if len(a) == 2 {
					// Special -- arg
					for _, a := range argv {
						p.addFile(a, DetectFormat(a))
					}
					return
				}
				name, value := splitArgV(a[2:])
				argv, err = p.parseLong(name, value, argv)
			default:
				argv, err = p.parseShort(a[1:], argv)
			}
		} else {
			p.addFile(a, Auto)
		}
	}
	return
}
func (p *argParser) Tasks() (tasks []StreamTask) {
	tasks = append(tasks, p.tasks...)
	if len(tasks) == 0 {
		tasks = append(tasks, p.inputTask())
	}

	return append(tasks, StreamWriteTo(os.Stdout, p.output))
}

func (p *argParser) inputTask() (s StreamTaskSequence) {
	if p.input == nil {
		return append(s, ReadFromTask(os.Stdin, YAML))
	}
	s = append(s, p.input...)
	p.input = p.input[:0]
	return
}
