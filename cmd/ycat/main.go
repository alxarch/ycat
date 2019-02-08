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
	args := ycat.Arguments{}
	if err := args.Parse(os.Args[1:]); err != nil {
		printUsage(err)
		os.Exit(2)
	}
	if args.Help {
		printUsage(nil)
		os.Exit(0)
	}

	p := args.Run(context.Background())
	exitCode := 0
	for err := range p.Errors() {
		if err != nil {
			exitCode = 2
			logger.Println(err)
		}
	}
	os.Exit(exitCode)
}
