package command

import (
	"fmt"
	"os"
	"syscall"

	"github.com/alpacanetworks/alpamon-go/pkg/collector"
	"github.com/alpacanetworks/alpamon-go/pkg/config"
	"github.com/alpacanetworks/alpamon-go/pkg/db"
	"github.com/alpacanetworks/alpamon-go/pkg/logger"
	"github.com/alpacanetworks/alpamon-go/pkg/pidfile"
	"github.com/alpacanetworks/alpamon-go/pkg/runner"
	"github.com/alpacanetworks/alpamon-go/pkg/scheduler"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/alpacanetworks/alpamon-go/pkg/version"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "alpamon",
	Short: "Secure Server Agent for Alpacon",
	Run: func(cmd *cobra.Command, args []string) {
		runAgent()
	},
}

func init() {
	RootCmd.AddCommand(setupCmd, ftpCmd)
}

func runAgent() {
	// platform
	utils.InitPlatform()

	// Pid
	pidFilePath, err := pidfile.WritePID(pidfile.FilePath("alpamon"))
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to create PID file", err.Error())
		os.Exit(1)
	}
	defer func() { _ = os.Remove(pidFilePath) }()

	fmt.Printf("alpamon version %s starting.\n", version.Version)

	// Config & Settings
	settings := config.LoadConfig()
	config.InitSettings(settings)

	// Session
	session := scheduler.InitSession()
	commissioned := session.CheckSession()

	// Reporter
	scheduler.StartReporters(session)

	// Logger
	logFile := logger.InitLogger()
	defer func() { _ = logFile.Close() }()
	log.Info().Msg("alpamon initialized and running.")

	// Commit
	runner.CommitAsync(session, commissioned)

	// DB
	client := db.InitDB()

	// Collector
	metricCollector := collector.InitCollector(session, client)
	metricCollector.Start()
	defer metricCollector.Stop()

	// Websocket Client
	wsClient := runner.NewWebsocketClient(session)
	wsClient.RunForever()

	if wsClient.RestartRequested {
		if err = os.Remove(pidFilePath); err != nil {
			log.Error().Err(err).Msg("Failed to remove PID file")
			return
		}

		executable, err := os.Executable()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get executable path")
			return
		}

		args := os.Args

		err = syscall.Exec(executable, args, os.Environ())
		if err != nil {
			log.Error().Err(err).Msg("Failed to restart the program")
		}
	}

	log.Debug().Msg("Bye.")
}
