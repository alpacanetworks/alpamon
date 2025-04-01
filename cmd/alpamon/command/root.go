package command

import (
	"context"
	"fmt"
	"github.com/alpacanetworks/alpamon/cmd/alpamon/command/ftp"
	"github.com/alpacanetworks/alpamon/cmd/alpamon/command/setup"
	"github.com/alpacanetworks/alpamon/pkg/collector"
	"github.com/alpacanetworks/alpamon/pkg/config"
	"github.com/alpacanetworks/alpamon/pkg/db"
	"github.com/alpacanetworks/alpamon/pkg/logger"
	"github.com/alpacanetworks/alpamon/pkg/pidfile"
	"github.com/alpacanetworks/alpamon/pkg/runner"
	"github.com/alpacanetworks/alpamon/pkg/scheduler"
	"github.com/alpacanetworks/alpamon/pkg/utils"
	"github.com/alpacanetworks/alpamon/pkg/version"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

const (
	name   = "alpamon"
	wsPath = "/ws/servers/backhaul/"
)

var RootCmd = &cobra.Command{
	Use:   "alpamon",
	Short: "Secure Server Agent for Alpacon",
	Run: func(cmd *cobra.Command, args []string) {
		runAgent()
	},
}

func init() {
	setup.SetConfigPaths(name)
	RootCmd.AddCommand(setup.SetupCmd, ftp.FtpCmd)
}

func runAgent() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			cancel()
		}
	}()

	// platform
	utils.InitPlatform()

	// Pid
	pidFilePath, err := pidfile.WritePID(pidfile.FilePath(name))
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to create PID file", err.Error())
		os.Exit(1)
	}

	fmt.Printf("alpamon version %s starting.\n", version.Version)

	// Config & Settings
	settings := config.LoadConfig(config.Files(name), wsPath)
	config.InitSettings(settings)

	// Session
	session := scheduler.InitSession()
	commissioned := session.CheckSession(ctx)

	// Reporter
	scheduler.StartReporters(session)

	// Logger
	logFile := logger.InitLogger()
	log.Info().Msg("alpamon initialized and running.")

	// Commit
	runner.CommitAsync(session, commissioned)

	// DB
	client := db.InitDB()

	// Collector
	metricCollector := collector.InitCollector(session, client)
	metricCollector.Start()

	// Websocket Client
	wsClient := runner.NewWebsocketClient(session)
	go wsClient.RunForever()

	select {
	case <-ctx.Done():
		log.Info().Msg("Received termination signal. Shutting down...")
		break
	case <-wsClient.ShutDownChan:
		log.Info().Msg("Shutdown command received. Shutting down...")
		break
	case <-wsClient.RestartChan:
		log.Info().Msg("Restart requested internally.")
		gracefulShutdown(metricCollector, wsClient, logFile, pidFilePath)
		restartAgent()
		return
	}

	gracefulShutdown(metricCollector, wsClient, logFile, pidFilePath)
}

func restartAgent() {
	executable, err := os.Executable()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get executable path")
		return
	}

	err = syscall.Exec(executable, os.Args, os.Environ())
	if err != nil {
		log.Error().Err(err).Msg("Failed to restart the program")
	}
}

func gracefulShutdown(collector *collector.Collector, wsClient *runner.WebsocketClient, logFile *os.File, pidPath string) {
	if collector != nil {
		collector.Stop()
	}
	if wsClient != nil {
		wsClient.Close()
	}
	log.Debug().Msg("Bye.")
	if logFile != nil {
		_ = logFile.Close()
	}
	_ = os.Remove(pidPath)
}
