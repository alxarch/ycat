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

	ok := true
	for err := range args.Run(context.Background()) {
		if err != nil {
			ok = false
			logger.Println(err)
		}
	}
	if !ok {
		os.Exit(2)
	}
}
