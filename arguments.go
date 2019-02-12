package ycat

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	jsonnet "github.com/google/go-jsonnet"
)

// ParseArgs builds a StreamTask sequence from arguments
func ParseArgs(argv []string, stdin io.Reader, stdout io.WriteCloser) ([]StreamTask, bool, error) {
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}
	p := argParser{
		stdin:  stdin,
		stdout: stdout,
	}
	if err := p.Parse(argv); err != nil {
		return nil, false, err
	}
	return p.Tasks(), p.help, nil
}

type argParser struct {
	vm     *jsonnet.VM
	stdin  io.Reader
	stdout io.WriteCloser
	eval   Eval
	output Output
	input  Producers
	tasks  []StreamTask
	help   bool
	err    error
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
						p.addFile(a, Auto)
					}
					return
				}
				name, value := splitArgV(a[2:])
				argv, err = p.parseLong(name, value, argv)
			default:
				name, ok := shortArgs[c]
				if !ok {
					return fmt.Errorf("Invalid short option: -%c", c)
				}
				argv, err = p.parseLong(name, a[2:], argv)
			}
		} else {
			p.addFile(a, Auto)
		}
	}
	return
}
func (p *argParser) Tasks() (tasks []StreamTask) {
	tasks = append(tasks, p.tasks...)
	if task := p.inputTask(); task != nil {
		tasks = append(tasks, task)
	} else if len(tasks) == 0 {
		tasks = append(tasks, ReadFromTask(p.stdin, YAML))
	}
	tasks = append(tasks, p.outputTask())
	return tasks
}

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

var shortArgs = map[byte]string{
	'j': "json",
	'y': "yaml",
	'i': "import",
	'v': "var",
	'e': "eval",
	'o': "output",
	'n': "null",
	'a': "array",
	'h': "help",
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
		typ := CodeVar
		if len(value) > 0 && value[0] == '=' {
			value = value[1:]
			typ = RawVar
		}
		p.Eval().AddVar(typ, name, value)
	case "eval":
		value, argv = shiftArgV(value, argv)
		filename, err := EvalFilename()
		if err != nil {
			return argv, err
		}
		p.addTask(p.eval.Snippet(filename, value))
	case "output":
		value, argv = shiftArgV(value, argv)
		if p.output = OutputFromString(value); p.output == OutputInvalid {
			return argv, fmt.Errorf("Invalid output format: %q", value)
		}
	case "null":
		p.input = append(p.input, NullStream{})
	case "to-json":
		p.output = OutputJSON
	case "help":
		p.help = true
	case "array":
		p.addTask(ToArray{})
	case "yaml":
		return p.parseFiles(value, argv, YAML), nil
	case "json":
		return p.parseFiles(value, argv, JSON), nil
	case "debug":
		value, argv = shiftArgV(value, argv)
		if value == "" {
			value = "DEBUG"
		}
		p.addTask(Debug(value))
	default:
		return argv, fmt.Errorf("Invalid option: %q", name)
	}
	return argv, nil
}

func splitArgV(s string) (string, string) {
	if n := strings.IndexByte(s, '='); 0 <= n && n < len(s) {
		return s[:n], s[n+1:]
	}
	return s, ""
}

func (p *argParser) Eval() *Eval {
	return &p.eval

}

func (p *argParser) addTask(t StreamTask) {
	if input := p.inputTask(); input == nil {
		if len(p.tasks) == 0 {
			input = ReadFromTask(p.stdin, YAML)
			p.tasks = append(p.tasks, input, t)
		} else {
			p.tasks = append(p.tasks, t)
		}
	} else {
		p.tasks = append(p.tasks, input, t)
	}
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

func (p *argParser) addFile(path string, format Format) {
	if format == Auto {
		format = DetectFormat(path)
	}
	if format == JSONNET {
		p.addTask(p.eval.SnippetFromFile(path))
		return
	}
	switch path {
	case "", "-":
		// Handle here to be able to test stdin
		p.input = append(p.input, ReadFromTask(p.stdin, format))
	default:
		p.input = append(p.input, ReadFromFile(path, format))
	}
}

func (p *argParser) outputTask() (s StreamTask) {
	switch p.output {
	case OutputJSON:
		return StreamWriteJSON(p.stdout)
	default:
		return StreamWriteYAML(p.stdout)
	}
}

func (p *argParser) inputTask() (s StreamTask) {
	if p.input == nil {
		return nil
	}
	s = p.input
	p.input = nil
	return
}
