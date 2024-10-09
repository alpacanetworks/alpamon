package main

import (
	"os"

	"github.com/alpacanetworks/alpamon-go/cmd/alpamon/command"
)

func main() {
	if err := command.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
