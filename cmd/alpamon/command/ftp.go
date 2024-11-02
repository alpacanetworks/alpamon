package command

import (
	"github.com/alpacanetworks/alpamon-go/pkg/config"
	"github.com/alpacanetworks/alpamon-go/pkg/logger"
	"github.com/alpacanetworks/alpamon-go/pkg/runner"
	"github.com/spf13/cobra"
)

var ftpCmd = &cobra.Command{
	Use:   "ftp <url> <homeDirectory>",
	Short: "Start worker for Web FTP",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		data := runner.FtpConfigData{
			URL:           args[0],
			HomeDirectory: args[1],
			Logger:        logger.NewFtpLogger(),
			Settings:      config.LoadConfig(),
		}

		RunFtpWorker(data)
	},
}

func RunFtpWorker(data runner.FtpConfigData) {
	ftpClient := runner.NewFtpClient(data)
	ftpClient.RunFtpBackground()
}
