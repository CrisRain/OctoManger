package main

import (
	"fmt"
	"os"

	"octomanger/internal/platform/entrypoint"
)

var runEntrypoint = entrypoint.Run

func run(args []string) int {
	if err := runEntrypoint(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:]))
}
