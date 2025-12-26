// nuke - A safer, faster, and more user-friendly alternative to rm
// Main entry point for the CLI application
package main

import (
	"fmt"
	"os"

	"nuke/cmd"
)

// Version information - set by goreleaser at build time
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const banner = `
nuke - A Safer rm Alternative %s

         _ ._  _ , _ ._
        (_   (    )_  .__)
      ( (  (    )    )  ) _)
     (__ (_(_ . _ )_)__)
           |  |  |
           |  |  |
           |  |  |
           \  |  /
            \ | /
             \|/
              V  <-- (The Incoming Delivery)

               	 __
                / _)  - "Is it getting hot in here?"
       _.----._/ /
      /         /
   __/ (  | (  |
  /__.-'|_|--|_|

Developed by Aman Dhruva Thamminana
Feedback: thammina@msu.edu
Contribute: https://github.com/amantham20/nuke_rm
`

func main() {
	// Check for version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-V" || os.Args[1] == "version") {
		fmt.Printf(banner, version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
		return
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
