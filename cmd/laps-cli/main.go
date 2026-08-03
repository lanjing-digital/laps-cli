package main

import (
	"os"

	"production-scheduling-cli/internal/commands"
)

func main() {
	os.Exit(commands.Run(os.Args[1:], os.Stdout, os.Stderr))
}
