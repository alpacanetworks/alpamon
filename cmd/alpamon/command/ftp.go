package command

import (
	"github.com/alpacanetworks/alpamon-go/pkg/logger"
	"github.com/alpacanetworks/alpamon-go/pkg/runner"
	"github.com/spf13/cobra"
)

var ftpCmd = &cobra.Command{
	Use:   "ftp <url> <homeDirectory>",
	Short: "Start worker for Web FTP",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		homeDirectory := args[1]

		ftpLogger := logger.NewFtpLogger()

		RunFtpWorker(url, homeDirectory, ftpLogger)
	},
}

func RunFtpWorker(url, homeDirectory string, ftpLogger logger.FtpLogger) {
	ftpClient := runner.NewFtpClient(url, homeDirectory, ftpLogger)
	ftpClient.RunFtpBackground()
}
