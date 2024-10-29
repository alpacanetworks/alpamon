package command

import (
	"github.com/alpacanetworks/alpamon-go/pkg/config"
	"github.com/alpacanetworks/alpamon-go/pkg/runner"
	"github.com/spf13/cobra"
)

var ftpCmd = &cobra.Command{
	Use:   "ftp <url> <homeDirectory>",
	Short: "Start worker for Web FTP",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		homeDirectory := args[1]

		settings := config.LoadConfig()
		config.InitFtpSettings(settings)

		RunFtpWorker(url, homeDirectory)

		return nil
	},
}

func RunFtpWorker(url, homeDirectory string) {
	ftpClient := runner.NewFtpClient(url, homeDirectory)
	ftpClient.RunFtpBackground()
}
