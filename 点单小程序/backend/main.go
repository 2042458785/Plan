package main

import (
	"Boutique/src/cmd"
	"os"
)

func main() {
	defer cmd.Clean()
	if err := cmd.Start(); err != nil {
		cmd.Clean()
		os.Exit(1)
	}
}
