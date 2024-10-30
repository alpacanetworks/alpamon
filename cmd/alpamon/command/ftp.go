package command

import (
	"fmt"
	"os/user"
	"strconv"
	"syscall"

	"github.com/alpacanetworks/alpamon-go/pkg/config"
	"github.com/alpacanetworks/alpamon-go/pkg/logger"
	"github.com/alpacanetworks/alpamon-go/pkg/runner"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var ftpCmd = &cobra.Command{
	Use:   "ftp <username> <groupname> <url> <homeDirectory>",
	Short: "Start worker for Web FTP",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		groupname := args[1]
		url := args[2]
		homeDirectory := args[3]

		logFile := logger.InitLogger()
		defer func() { _ = logFile.Close() }()

		settings := config.LoadConfig()
		config.InitFtpSettings(settings)

		if syscall.Getuid() == 0 {
			if username == "" || groupname == "" {
				log.Debug().Msg("No username or groupname provided.")
				return fmt.Errorf("No username or groupname provided.")
			}

			usr, err := user.Lookup(username)
			if err != nil {
				log.Debug().Msgf("There is no corresponding %s username in this server", username)
				return fmt.Errorf("There is no corresponding %s username in this server", username)
			}

			group, err := user.LookupGroup(groupname)
			if err != nil {
				log.Debug().Msgf("There is no corresponding %s groupname in this server", groupname)
				return fmt.Errorf("There is no corresponding %s groupname in this server", groupname)
			}

			uid, err := strconv.Atoi(usr.Uid)
			if err != nil {
				return err
			}

			gid, err := strconv.Atoi(group.Gid)
			if err != nil {
				return err
			}

			err = syscall.Setgroups([]int{})
			if err != nil {
				return err
			}

			err = syscall.Setuid(uid)
			if err != nil {
				return err
			}

			err = syscall.Setgid(gid)
			if err != nil {
				return err
			}
		}

		RunFtpWorker(url, homeDirectory)

		return nil
	},
}

func RunFtpWorker(url, homeDirectory string) {
	ftpClient := runner.NewFtpClient(url, homeDirectory)
	ftpClient.RunFtpBackground()
}
