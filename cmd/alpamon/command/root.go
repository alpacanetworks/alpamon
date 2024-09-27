package command

import (
	"fmt"
	"github.com/alpacanetworks/alpamon-go/pkg/config"
	"github.com/alpacanetworks/alpamon-go/pkg/logger"
	"github.com/alpacanetworks/alpamon-go/pkg/pidfile"
	"github.com/alpacanetworks/alpamon-go/pkg/runner"
	"github.com/alpacanetworks/alpamon-go/pkg/scheduler"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/alpacanetworks/alpamon-go/pkg/version"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
)

var RootCmd = &cobra.Command{
	Use:   "alpamon",
	Short: "Secure Server Agent for Alpaca Infra Platform",
	Run: func(cmd *cobra.Command, args []string) {
		runAgent()
	},
}

func runAgent() {
	// platform
	utils.InitPlatform()

	// Pid
	pidFilePath, err := pidfile.WritePID()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create PID file", err.Error())
		os.Exit(1)
	}
	defer func() { _ = os.Remove(pidFilePath) }()

	// Logger
	logFile := logger.InitLogger()
	defer func() { _ = logFile.Close() }()

	// Config & Settings
	settings := config.LoadConfig()
	config.InitSettings(settings)
	fmt.Printf("alpamon-go %s starting.\n", version.Version)

	// Session
	session := scheduler.InitSession()
	commissioned := session.CheckSession()

	// Reporter
	scheduler.StartReporters(session)

	// Commit
	runner.CommitAsync(session, commissioned)

	// Websocket Client
	wsClient := runner.NewWebsocketClient(session)
	wsClient.RunForever()

	log.Debug().Msg("Bye.")
}
