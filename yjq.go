package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/alxarch/yjq/internal/yjq"
	"github.com/pborman/getopt/v2"
)

const (
	defaultCmd = "jq"
	jqEnvVar   = "JQ"
)

var (
	flags         = getopt.New()
	version       bool
	help          bool
	runTests      bool
	seqIO         bool
	streamIO      bool
	slurpInput    bool
	rawInput      bool
	nullInput     bool
	compactOutput bool
	tab           bool
	indent        int
	colorOutput   bool
	monoOutput    bool
	asciiOutput   bool
	unbuffered    bool
	sortKeys      bool
	rawOutput     bool
	joinOutput    bool
	filterFiles   *[]string
	searchDirs    *[]string
	exitStatus    bool
	strArgs       *[]string
	jsonArgs      *[]string
	slurpFiles    *[]string
	argFiles      *[]string
	rawFiles      *[]string
	yamlInput     bool
	yamlOutput    bool
	posArgs       bool
	posArgsJSON   bool
)

func init() {
	flags.FlagLong(&version, "version", 0, "Show version and exit")
	flags.FlagLong(&help, "help", 'h', "Show usage and exit")
	flags.FlagLong(&runTests, "run-tests", 0, "Run tests")
	flags.FlagLong(&seqIO, "seq", 0, "Use application/json-seq MIME for input and output")
	flags.FlagLong(&streamIO, "stream", 0, "Stream mod input and output")
	flags.FlagLong(&slurpInput, "slurp", 's',
		"Instead  of running the filter for each JSON object in the input, read the entire",
		"input stream into a large array and run the filter just once.")
	flags.FlagLong(&rawInput, "raw-input", 'R', "Each line of input is passed to the filter as a string")
	flags.FlagLong(&nullInput, "null-input", 'n', "Run the filter with `null` as input")
	flags.FlagLong(&compactOutput, "compact-output", 'c', "Output compact JSON one value per line")
	flags.FlagLong(&tab, "tab", 0, "Use tab for each indentation level")
	flags.FlagLong(&indent, "indent", 0, "Use the given number of spaces (no more than 8) for indentation")
	flags.FlagLong(&colorOutput, "color-output", 'C', "Force color output even if stdout is piped")
	flags.FlagLong(&monoOutput, "monochrome-output", 'M', "Force monochrome output even if stdout is a tty")
	flags.FlagLong(&asciiOutput, "ascii-output", 'a', "Escape unicode to ASCII")
	flags.FlagLong(&unbuffered, "unbuffered", 0, "Unbuffered output")
	flags.FlagLong(&sortKeys, "sort-keys", 'S', "Sort object keys")
	flags.FlagLong(&rawOutput, "raw-output", 'r', "Output string results without quotes")
	flags.FlagLong(&joinOutput, "join-output", 'j', "Concatenate string results without quotes")
	filterFiles = flags.ListLong("from-file", 'f', "Read filter from file")
	searchDirs = flags.List('L', "Prepend dir to the search list for modules")
	flags.FlagLong(&exitStatus, "exit-status", 'e', "Set exit status to 1 if last result was false or null or 4 if no result was produced")
	strArgs = flags.ListLong("arg", 0, "Pass string arguments to the filter")
	jsonArgs = flags.ListLong("argjson", 0, "Pass JSON arguments to the filter")
	slurpFiles = flags.ListLong("slurpfile", 0, "set variable $a to an array of JSON texts read from <f>")
	rawFiles = flags.ListLong("rawfile", 0, "set variable $a to a string consisting of the contents of <f>")
	argFiles = flags.ListLong("argfile", 0, "set variable $a to an array of JSON texts read from <f>")
	flags.FlagLong(&posArgs, "args", 0, "Remaining arguments are positional string arguments.")
	flags.FlagLong(&posArgsJSON, "jsonargs", 0, "Remaining arguments are positional JSON arguments.")
	flags.FlagLong(&yamlInput, "yaml-input", 'Y', "Process input as YAML")
	flags.FlagLong(&yamlOutput, "yaml-output", 'y', "Output results as YAML")
	flags.SetUsage(usage)
}

func main() {

	logger := log.New(os.Stderr, "yjq: ", 0)

	filter, inputFiles, positionalArgs, err := parseArgs(os.Args)
	if err != nil {
		logger.Println(err)
		usage()
		os.Exit(2)
	}

	if hasYAMLInputFile(inputFiles) {
		yamlInput = true
	}

	sanitizeFlags()

	args := rewriteArgs()
	if filter != "" {
		args = append(args, filter)
	}
	args = append(args, positionalArgs...)

	cmdName := os.Getenv(jqEnvVar)
	if cmdName == "" {
		cmdName = defaultCmd
	}
	cmd := exec.Command(cmdName, args...)
	cmd.Stderr = os.Stderr
	switch {
	case version, help, runTests, !(yamlInput || yamlOutput):
		// Passthrough
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Start()
		os.Exit(onExit(cmd.Wait()))
	}
	wg := new(sync.WaitGroup)
	var (
		jqw io.WriteCloser
		jqr io.ReadCloser
	)
	if yamlInput {
		if jqw, err = cmd.StdinPipe(); err != nil {
			logger.Printf("Failed to open jq stdin: %s", err)
			os.Exit(1)
		}
	} else {
		cmd.Stdin = os.Stdin
	}
	if yamlOutput {
		if jqr, err = cmd.StdoutPipe(); err != nil {
			logger.Printf("Failed to open jq stdout: %s", err)
			os.Exit(1)
		}
	} else {
		cmd.Stdout = os.Stdout
	}

	if err = cmd.Start(); err != nil {
		logger.Printf("Failed to run jq: %s", err)
		os.Exit(onExit(err))
	}

	if yamlInput {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer jqw.Close()
			if len(inputFiles) > 0 {
				for _, filename := range inputFiles {
					if err := yjq.CopyFile(filename, jqw); err != nil {
						logger.Printf("Failed to read file %q: %s", filename, err)
						os.Exit(1)
					}
				}
			} else {
				if err := yjq.CopyYAMLToJSON(jqw, os.Stdin); err != nil {
					logger.Printf("Failed to read stdin: %s", err)
					os.Exit(1)
				}
			}
		}()
	}

	if yamlOutput {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer jqr.Close()
			if _, err := yjq.CopyJSONToYAML(os.Stdout, jqr); err != nil {
				logger.Printf("Failed to write to stdout: %s", err)
				os.Exit(1)
			}
		}()
		wg.Wait()
	}

	if err = cmd.Wait(); err != nil {
		logger.Printf("jq error: %s", err)
		os.Exit(onExit(err))
	}
}

func onExit(err error) (exitCode int) {
	if err, ok := err.(*exec.ExitError); ok {
		if status, ok := err.ProcessState.Sys().(syscall.WaitStatus); ok {
			return int(status)
		}
	}
	return 1
}

func indexOf(values []string, s string) int {
	for i := range values {
		if values[i] == s {
			return i
		}
	}
	return -1
}

func rewriteArgs() (args []string) {
	if help {
		return append(args, "--help")
	}
	if version {
		return append(args, "--version")
	}
	if runTests {
		args = append(args, "--run-tests")
		if testfile := flags.Arg(0); testfile != "" {
			args = append(args, testfile)
		}
		return
	}
	if seqIO {
		args = append(args, "--seq")
	}
	if streamIO {
		args = append(args, "--stream")
	}
	if slurpInput {
		args = append(args, "--slurp")
	}
	if rawInput {
		args = append(args, "--raw-input")
	}
	if nullInput {
		args = append(args, "--null-input")
	}
	if compactOutput {
		args = append(args, "--compact-output")
	}
	if tab {
		args = append(args, "--tab")
	}
	if indent > 0 {
		args = append(args, "--indent", strconv.Itoa(indent))
	}
	if colorOutput {
		args = append(args, "--color-output")
	}
	if monoOutput {
		args = append(args, "--monochrome-output")
	}
	if asciiOutput {
		args = append(args, "--ascii-output")
	}
	if unbuffered {
		args = append(args, "--unbuffered")
	}
	if sortKeys {
		args = append(args, "--sort-keys")
	}
	if rawOutput {
		args = append(args, "--raw-output")
	}
	if joinOutput {
		args = append(args, "--join-output")
	}
	if exitStatus {
		args = append(args, "--exit-status")
	}
	if len(*searchDirs) > 0 {
		for _, f := range *searchDirs {
			args = append(args, "-L", f)
		}
	}
	args = appendPairArgs(args, "--arg", *strArgs)
	args = appendPairArgs(args, "--argjson", *jsonArgs)
	args = appendPairArgs(args, "--slurpfile", *slurpFiles)
	args = appendPairArgs(args, "--rawfile", *rawFiles)
	args = appendPairArgs(args, "--argfile", *argFiles)
	for _, f := range *filterFiles {
		args = append(args, "--from-file", f)
	}
	return
}

func appendPairArgs(args []string, opt string, pairs []string) []string {
	for _, s := range pairs {
		if pair := strings.SplitN(s, "=", 2); len(pair) == 2 {
			args = append(args, opt, pair[0], pair[1])
		}
	}
	return args

}

func positionalArgsIndex(args []string) int {
	for i, a := range args {
		switch a {
		case "--jsonargs", "--args":
			return i
		}
	}
	return -1

}
func preProcessArgs(args []string) (out []string) {
	// Pre process args to accomodate jq's weird double-value arguments
	var pairArgs = []string{
		"--arg",
		"--argjson",
		"--slurpfile",
		"--rawfile",
		"--argfile",
	}

	for len(args) > 0 {
		arg := args[0]
		args = args[1:]
		out = append(out, arg)
		if arg == "--" {
			return append(out, args...)
		}
		if indexOf(pairArgs, arg) != -1 {
			if len(args) > 0 {
				arg = args[0]
				args = args[1:]
				if strings.IndexByte(arg, '=') == -1 {
					if len(args) > 0 {
						arg = arg + "=" + args[0]
						args = args[1:]
					} else {
						arg = arg + "="
					}
				}
				out = append(out, arg)
			}
		}
	}
	return
}

func hasYAMLInputFile(inputFiles []string) bool {
	for _, f := range inputFiles {
		switch path.Ext(f) {
		case ".yaml", ".yml":
			return true
		}
	}
	return false
}

func sanitizeFlags() {
	if nullInput {
		yamlInput = false
	}
	if yamlOutput {
		seqIO = false
		colorOutput = false
		rawOutput = false
		joinOutput = false
		monoOutput = true
		compactOutput = true
		unbuffered = true
		tab = false
		indent = 0
	}
}

func parseArgs(args []string) (filter string, inputFiles, positionalArgs []string, err error) {
	args = preProcessArgs(args)
	if offset := positionalArgsIndex(args); offset >= 0 {
		positionalArgs = args[offset:]
		args = args[offset:]
	}
	if err = flags.Getopt(args, nil); err != nil {
		return
	}

	args = flags.Args()
	if len(*filterFiles) == 0 {
		if len(args) > 0 {
			inputFiles = args[1:]
			args = args[:1]
		} else {
			if len(positionalArgs) > 0 {
				args = positionalArgs[:1]
				positionalArgs = positionalArgs[1:]
			}
		}
	} else {
		inputFiles = append(inputFiles, args...)
		args = args[:0]
	}
	if len(args) > 0 {
		filter = args[0]
		args = args[1:]
	}
	return
}

func usage() {
	fmt.Fprintln(os.Stderr, `
yjq - YAML wrapper for jq commandline JSON processor

Usage: yjq [options] <jq filter> [file...]
       yjq [options] --args <jq filter> [strings...]
       yjq [options] --jsonargs <jq filter> [JSON_TEXTS...]

Options:`)
	flags.PrintOptions(os.Stderr)
}
