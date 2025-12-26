// nuke - A safer, faster, and more user-friendly alternative to rm
// Main entry point for the CLI application
package main

import (
	"fmt"
	"os"

	"nuke/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
