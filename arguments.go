package ycat

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Usage for ycat cmd
const Usage = `
ycat - command line YAML/JSON processor

Usage: ycat [options|files...]

Options:
    -h, --help                   Show help and exit
    -y, --yaml [files...]        Read YAML values from file(s)
    -j, --json [files...]        Read JSON values from file(s)
    -n, --null                   Use null value input (no reading)
    -o, --out {json|j|yaml|y}    Set output format
        --to-json                Output JSON one value per line (same as -o json, -oj)
    -a, --array                  Merge values into an array
    -e, --eval <snippet>         Process values with Jsonnet
    -v, --var <var>=<code>       Bind Jsonnet variable to code
              <var>==<value>     Bind Jsonnet variable to a string value
    -i, --import <var>=<file>    Import file into a local Jsonnet variable
        --input-var <var>        Change the name of the input value variable (default x) 
        --max-stack <size>       Jsonnet VM max stack size (default 500)

If no files are specified values are read from stdin.
Using "-" as a file path will read values from stdin.
Files without a format option will be parsed as YAML unless
they end in ".json".
`

type Arguments struct {
	Help   bool
	Output Output
	Eval   Eval
	Array  bool
	Null   bool
	Files  []InputFile
}

func (args *Arguments) Parse(argv []string) (err error) {
	for err == nil && len(argv) > 0 {
		a := argv[0]
		argv = argv[1:]
		if len(a) > 1 && a[0] == '-' {
			switch c := a[1]; c {
			case '-':
				if len(a) == 2 {
					// Special -- arg
					for _, a := range argv {
						args.addFile(a, Auto)
					}
					return
				}
				name, value := splitArgV(a[2:])
				argv, err = args.parseLong(name, value, argv)
			default:
				argv, err = args.parseShort(a[1:], argv)
			}
		} else {
			args.addFile(a, Auto)
		}

	}
	return
}

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

func (args *Arguments) parseLong(name, value string, argv []string) ([]string, error) {
	switch name {
	case "max-stack":
		value, argv = shiftArgV(value, argv)
		size, err := strconv.Atoi(value)
		if err != nil {
			return argv, fmt.Errorf("Invalid max stack size: %s", err)
		}
		args.Eval.MaxStackSize = size
	case "input-var":
		value, argv = shiftArgV(value, argv)
		args.Eval.Bind = value
	case "import":
		value, argv = shiftArgV(value, argv)
		name, file := splitArgV(value)
		if file == "" {
			return argv, errors.New("Missing import file")
		}
		args.Eval.AddVar(FileVar, name, file)
	case "var":
		value, argv = shiftArgV(value, argv)
		name, value := splitArgV(value)
		typ := RawVar
		if len(value) > 0 && value[0] == '=' {
			value = value[1:]
			typ = CodeVar
		}
		args.Eval.AddVar(typ, name, value)
	case "eval":
		if value, argv = shiftArgV(value, argv); value == "--" {
			value = strings.Join(argv, " ")
			argv = nil
		}
		args.Eval.Snippet = value
	case "output":
		value, argv = shiftArgV(value, argv)
		if args.Output = OutputFromString(value); args.Output == OutputInvalid {
			return argv, fmt.Errorf("Invalid output format: %q", value)
		}
	case "null":
		args.Null = true
	case "to-json":
		args.Output = OutputJSON
	case "help":
		args.Help = true
	case "array":
		if args.Eval.Snippet == "" {
			args.Array = true
		} else {
			args.Eval.Array = true
		}
	case "yaml":
		return args.parseFiles(value, argv, YAML), nil
	case "json":
		return args.parseFiles(value, argv, JSON), nil
	default:
		return argv, fmt.Errorf("Invalid option: %q", name)
	}
	return argv, nil
}

func (args *Arguments) parseShort(a string, argv []string) ([]string, error) {
	for ; len(a) > 0; a = a[1:] {
		switch c := a[0]; c {
		case 'j':
			return args.parseFiles(a[1:], argv, JSON), nil
		case 'y':
			return args.parseFiles(a[1:], argv, YAML), nil
		case 'i':
			return args.parseLong("import", a[1:], argv)
		case 'v':
			return args.parseLong("var", a[1:], argv)
		case 'e':
			return args.parseLong("eval", a[1:], argv)
		case 'o':
			return args.parseLong("output", a[1:], argv)
		case 'n':
			args.Null = true
		case 'a':
			args.Array = true
		case 'h':
			args.Help = true
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

func (args *Arguments) addFile(path string, format Format) {
	args.Files = append(args.Files, InputFile{
		Format: format,
		Path:   path,
	})
}

func (args *Arguments) parseFiles(path string, argv []string, format Format) []string {
	switch {
	case len(path) > 0:
		if path[0] == '=' {
			path = path[1:]
		}
		args.addFile(path, format)
	case len(argv) == 0:
		args.addFile("", format)
	default:
		for ; len(argv) > 0 && !isOption(argv[0]); argv = argv[1:] {
			args.addFile(argv[0], format)
		}
	}
	return argv
}

func isOption(a string) bool {
	return len(a) > 1 && a[0] == '-' && (a[1] != '-' || len(a) > 2)
}

func withCancel(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithCancel(ctx)
}
func (args *Arguments) Run(ctx context.Context) Pipeline {

	var (
		tasks []StreamTask
		input []StreamTask
	)

	if args.Null {
		input = append(input, StreamFunc(NullStream))
	} else if len(args.Files) == 0 {
		input = append(input, ReadFromTask(os.Stdin, YAML))
	} else {
		for i := range args.Files {
			input = append(input, &args.Files[i])
		}
	}

	tasks = append(tasks, StreamTaskSequence(input))

	if args.Array {
		tasks = append(tasks, StreamFunc(StreamToArray))
	}

	if eval := args.Eval.StreamTask(); eval != nil {
		tasks = append(tasks, eval)
		if args.Eval.Array {
			tasks = append(tasks, StreamFunc(StreamToArray))
		}
	}

	switch args.Output {
	case OutputJSON:
		tasks = append(tasks, StreamWriteTo(os.Stdout, JSON))
	default:
		tasks = append(tasks, StreamWriteTo(os.Stdout, YAML))
	}

	return BlankPipeline().Pipe(ctx, tasks...)

}
