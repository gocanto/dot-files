package app

import "fmt"

func (a app) usage() {
	fmt.Fprintln(a.stdout, `mac-os manages this machine's dotfiles, developer tools, and macOS settings.

Usage:
  mac-os serve-http --socket <path> [settings flags]
  mac-os list-workflows
  mac-os run-workflow <id> [--preview]

The Electron app starts the HTTP backend to display workflows, execute runs, and read persisted logs.
The CLI run-workflow subcommand executes the same phases in a terminal so failures are visible directly.`)
}
