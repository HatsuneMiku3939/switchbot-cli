package main

import (
	"os"

	"github.com/hatsunemiku3939/switchbot-cli/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, os.Environ()))
}
