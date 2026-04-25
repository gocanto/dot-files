package main

import (
	"os"

	"github.com/gocanto/mac-os/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:]))
}
