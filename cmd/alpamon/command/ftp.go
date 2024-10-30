package command

import (
	"github.com/alpacanetworks/alpamon-go/pkg/config"
	"github.com/alpacanetworks/alpamon-go/pkg/logger"
	"github.com/alpacanetworks/alpamon-go/pkg/runner"
	"github.com/spf13/cobra"
)

var ftpCmd = &cobra.Command{
	Use:   "ftp <username> <groupname> <url> <homeDirectory>",
	Short: "Start worker for Web FTP",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		data := runner.CommandData{
			Username:      args[0],
			Groupname:     args[1],
			URL:           args[2],
			HomeDirectory: args[3],
		}

		logFile := logger.InitLogger()
		defer func() { _ = logFile.Close() }()

		settings := config.LoadConfig()
		config.InitFtpSettings(settings)

		RunFtpWorker(data)

		return nil
	},
}

func RunFtpWorker(data runner.CommandData) {
	ftpClient := runner.NewFtpClient(data)
	ftpClient.RunFtpBackground()
}
