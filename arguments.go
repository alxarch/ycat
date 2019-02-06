package ycat

import (
	"context"
	"fmt"
	"os"
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
    -o, --out {json|j|yaml|y}    Set output format
        --to-json                Output JSON one value per line (same as -o json, -oj)
    -a, --array                  Merge output values into an array
    -e, --eval <snippet>         Process values with Jsonnet
    -m, --module <var>=<file>    Load Jsonnet module into a local Jsonnet variable
        --bind <var>             Bind input value to a local Jsonnet variable (default _)

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
		return nil, nil
	case "bind":
		value, argv = shiftArgV(value, argv)
		args.Eval.Bind = value
	case "module":
		value, argv = shiftArgV(value, argv)
		name, file := splitArgV(value)
		if file == "" {
			return argv, fmt.Errorf("Invalid module value: %q", value)
		}
		args.Eval.AddModule(name, file)
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
	case "to-json":
		args.Output = OutputJSON
	case "help":
		args.Help = true
	case "array":
		args.Array = true
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
		case 'm':
			return args.parseLong("module", a[1:], argv)
		case 'e':
			return args.parseLong("eval", a[1:], argv)
		case 'o':
			return args.parseLong("output", a[1:], argv)
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

func (args *Arguments) Run(ctx context.Context) <-chan error {
	ctx, cancel := withCancel(ctx)
	defer cancel()

	var steps []PipelineFunc
	if len(args.Files) == 0 {
		steps = append(steps, ReadFrom(os.Stdin, YAML))
	} else {
		steps = append(steps, ReadFiles(args.Files...))
	}

	if eval, err := args.Eval.Pipeline(); err != nil {
		return wrapErr(err)
	} else if eval != nil {
		steps = append(steps, eval)
	}

	if args.Array {
		steps = append(steps, ToArray)
	}

	switch args.Output {
	case OutputJSON:
		steps = append(steps, WriteTo(os.Stdout, JSON))
	default:
		steps = append(steps, WriteTo(os.Stdout, YAML))
	}

	_, errc := BuildPipeline(ctx, steps...)
	return errc

}
