package app

import "fmt"

func (a app) usage() {
	fmt.Fprintln(a.stdout, `mac-os manages this machine's dotfiles, developer tools, and macOS settings.

Usage:
  mac-os
  mac-os tui

Commands:
  tui        Open the interactive Bubble Tea workflow dashboard.

Running mac-os with no arguments opens the same dashboard.`)
}
