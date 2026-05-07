package main

import (
	"context"
	"os"

	"github.com/gocanto/dot-files/tools/internal/reporoot"
	"github.com/gocanto/dot-files/tools/internal/turbocache"
)

func main() {
	rootDir, err := reporoot.Find("")

	if err != nil {
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}

	tool := turbocache.Tool{RootDir: rootDir}
	os.Exit(tool.Run(context.Background(), trimArgumentSeparator(os.Args[1:])))
}

func trimArgumentSeparator(args []string) []string {
	if len(args) > 0 && args[0] == "--" {
		return args[1:]
	}

	return args
}
