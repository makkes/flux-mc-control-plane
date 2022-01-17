package main

import (
	"os"

	"github.com/makkes/fluxmc/cmd"
)

func main() {
	if err := cmd.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
