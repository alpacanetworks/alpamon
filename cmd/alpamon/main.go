package main

import (
	"fmt"
	"github.com/alpacanetworks/alpamon-go/cmd/alpamon/command"
	"os"
)

func main() {
	if err := command.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
