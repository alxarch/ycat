package main

import (
	"context"
	"log"
	"os"

	"github.com/alxarch/ycat"
)

var (
	logger = log.New(os.Stderr, "ycat: ", 0)
)

func usage() {
	os.Stderr.WriteString(ycat.Usage)
}

func printUsage(err error) {
	if err != nil {
		logger.Println(err)
	}
	usage()
}

func main() {
	tasks, help, err := ycat.ParseArgs(os.Args[1:])
	if err != nil {
		printUsage(err)
		os.Exit(2)
	}
	if help {
		printUsage(nil)
		os.Exit(0)
	}

	ctx := context.Background()
	p := ycat.BlankPipeline().Pipe(ctx, tasks...)
	exitCode := 0
	for err := range p.Errors() {
		if err != nil {
			exitCode = 2
			logger.Println(err)
		}
	}
	os.Exit(exitCode)
}
