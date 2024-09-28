package main

import (
	"github.com/alpacanetworks/alpamon-go/cmd/alpamon/command"
	"os"
)

func main() {
	if err := command.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
