package app

import "fmt"

func (a app) usage() {
	fmt.Fprintln(a.stdout, `mac-os manages this machine's dotfiles, developer tools, and macOS settings.

Usage:
  mac-os serve-grpc --socket <path>

The Electron app starts this local gRPC backend to display workflows, execute runs, and read persisted logs.`)
}
