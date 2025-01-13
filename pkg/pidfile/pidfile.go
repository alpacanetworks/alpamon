package pidfile

import (
	"fmt"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	pidFilePathDarwin  = "/tmp/alpamon.pid"
	pidFilePathDefault = "/var/run/alpamon.pid"
)

// WritePID writes the current PID to a file, ensuring that the file
// doesn't exist or doesn't contain a PID for a running process.
//
// Based on a function from the Datadog Agent.
// Reference : https://github.com/DataDog/datadog-agent
func WritePID() (string, error) {
	pidFilePath, err := setupPIDFilePath()
	if err != nil {
		return "", err
	}

	// check whether the pidfile exists and contains the PID for a running proc...
	if byteContent, err := os.ReadFile(pidFilePath); err == nil {
		pidStr := strings.TrimSpace(string(byteContent))
		pid, err := strconv.Atoi(pidStr)
		if err == nil && isProcess(pid) {
			return "", fmt.Errorf("pidfile already exists, please check %s isn't running or remove %s",
				os.Args[0], pidFilePath)
		}
	}

	// create the full path to the pidfile
	if err := os.MkdirAll(filepath.Dir(pidFilePath), os.FileMode(0755)); err != nil {
		return "", err
	}

	// write current pid in it
	pidStr := fmt.Sprintf("%d", os.Getpid())
	if err := os.WriteFile(pidFilePath, []byte(pidStr), 0644); err != nil {
		return "", err
	}

	return pidFilePath, nil
}

// isProcess uses `kill -0` to check whether a process is running
func isProcess(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

func setupPIDFilePath() (string, error) {
	var pidFilePath string

	switch utils.PlatformLike {
	case "darwin":
		pidFilePath = pidFilePathDarwin
	default:
		pidFilePath = pidFilePathDefault
	}

	return pidFilePath, nil
}
