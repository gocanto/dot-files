package app

import "fmt"

func (a app) usage() {
	fmt.Fprintln(a.stdout, `mac-os manages this machine's dotfiles, developer tools, and macOS settings.

Usage:
  mac-os ui workflows
  mac-os ui run
  mac-os ui runs
  mac-os ui run-log --run-id <id>

The Electron app uses these JSON commands to display workflows, execute runs, and read persisted logs.`)
}
