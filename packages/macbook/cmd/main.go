package main

import (
	"os"

	"github.com/gocanto/dot-files/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:]))
}
