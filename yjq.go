package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/alxarch/yjq/internal/yjq"
	"github.com/spf13/pflag"
)

var (
	logger  = log.New(os.Stderr, "yjq: ", 0)
	flags   = pflag.NewFlagSet("yjq", pflag.ContinueOnError)
	cmdName = flags.String("cmd", "jq", "Name of the jq command")
	input   = flags.BoolP("yaml-input", "i", false, "Convert YAML input to JSON")
	output  = flags.BoolP("yaml-output", "o", false, "Convert JSON output to YAML")
	files   []string
	cmdArgs []string
	args    []string
)

func init() {
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, `
yjq - YAML wrapper for jq commandline JSON processor

Usage: yjq [options] [YAML_FILES...] -- [JQ_ARG...]
       yjq [JQ_ARG...]

Options:`)
		flags.PrintDefaults()
	}
}

func main() {
	flags.ParseErrorsWhitelist.UnknownFlags = true
	args = os.Args[1:]
	if hasArg(args, "--help", "-h") {
		flags.Usage()
		os.Exit(0)
	}
	if n := indexOf(args, "--"); n == -1 {
		cmdArgs = args
		args = args[:0]
	} else {
		cmdArgs = args[n+1:]
		args = args[:n]
	}
	err := flags.Parse(args)
	if err != nil {
		logger.Println(err)
		flags.Usage()
		os.Exit(2)
	}
	if files = flags.Args(); len(files) > 0 {
		*input = true
	}
	tty, err := isStdinTTY()
	if err != nil {
		logger.Println(err)
		os.Exit(2)
	}

	if *input == *output {
		*input = true
		*output = true
	}

	cmdArgs, *input, *output = rewriteArgs(cmdArgs, *input, *output)
	cmd := exec.Command(*cmdName, cmdArgs...)
	cmd.Stderr = os.Stderr
	wg := new(sync.WaitGroup)
	var (
		jqw io.WriteCloser
		jqr io.ReadCloser
	)
	if *input {
		if jqw, err = cmd.StdinPipe(); err != nil {
			onExit(err)
		}
	} else {
		cmd.Stdin = os.Stdin
	}
	if *output {
		if jqr, err = cmd.StdoutPipe(); err != nil {
			onExit(err)
		}
	} else {
		cmd.Stdout = os.Stdout
	}

	if err := cmd.Start(); err != nil {
		onExit(err)
	}

	if *input {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer jqw.Close()
			if tty && len(files) > 0 {
				for _, filename := range files {
					f, err := os.Open(filename)
					if err != nil {
						logger.Printf("Failed to open file %q: %s\n", filename, err)
						os.Exit(2)
					}
					defer f.Close()
					if err := yjq.CopyYAMLToJSON(jqw, f); err != nil {
						logger.Printf("Failed to parse file %q: %s\n", filename, err)
						os.Exit(2)
					}
				}
			} else {
				if err := yjq.CopyYAMLToJSON(jqw, os.Stdin); err != nil {
					logger.Println("Failed to read input", err)
					os.Exit(1)
				}
			}
		}()
	}

	if *output {
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

	if err := cmd.Wait(); err != nil {
		onExit(err)
	}
}

func exitCode(err error) int {
	if err, ok := err.(*exec.ExitError); ok {
		if code, ok := err.ProcessState.Sys().(syscall.WaitStatus); ok {
			return int(code)
		}
	}
	return 2
}

func onExit(err error) {
	switch code := exitCode(err); code {
	case 2, 3:
		logger.Println(err)
		fallthrough
	default:
		os.Exit(code)
	}
}

func hasArg(args []string, s ...string) bool {
	for _, s := range s {
		if indexOf(args, s) != -1 {
			return true
		}
	}
	return false
}
func omitArgs(args []string, omit ...string) (out []string) {
	for _, a := range args {
		if indexOf(omit, a) == -1 {
			out = append(out, a)
		}
	}
	return
}
func indexOf(values []string, s string) int {
	for i := range values {
		if values[i] == s {
			return i
		}
	}
	return -1
}

func injectArgs(args []string, inject ...string) (out []string) {
	out = append(out, inject...)
	for _, a := range args {
		if indexOf(inject, a) == -1 {
			out = append(out, a)
		}
	}
	return
}

func rewriteArgs(args []string, input, output bool) ([]string, bool, bool) {
	switch {
	case hasArg(args, "--help", "-h"):
		return []string{"--help"}, false, false
	case hasArg(args, "--version"):
		return []string{"--version"}, false, false
	case hasArg(args, "--run-tests"):
		return []string{"--run-tests"}, false, false
	}
	if hasArg(args, "--null-input", "-n") {
		input = false
	}
	if input {
		args = omitArgs(args, "--raw-input", "-R")
	}
	if output {
		args = omitArgs(args,
			"--color-output", "-C",
			"--tab",
			"--ascii-output", "-a",
			"--join-output", "-j",
			"--raw-output", "-r",
		)
		args = injectArgs(args, "--unbuffered", "--compact-output")
	}
	return args, input, output
}

func isStdinTTY() (bool, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return false, nil
	}
	return true, nil
}
