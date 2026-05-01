package app

import (
	"flag"
	"fmt"
)

func (a app) tui(args []string) int {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	if err := fs.Parse(args); err != nil {
		return 2
	}

	result, err := a.tuiRunner(a.stdin, a.stdout, a.tuiWorkflows())

	if err != nil {
		fmt.Fprintf(a.stderr, "tui failed: %v\n", err)

		return 1
	}

	return result.ExitCode
}
