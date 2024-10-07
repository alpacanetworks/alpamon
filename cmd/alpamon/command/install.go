package command

import (
	"embed"
	"fmt"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const (
	configTemplatePath = "configs/alpamon.conf"
	configTarget       = "/etc/alpamon/alpamon.conf"

	tmpFilePath   = "configs/tmpfile.conf"
	tmpFileTarget = "/usr/lib/tmpfiles.d/alpamon.conf"

	serviceTemplatePath = "configs/alpamon.service"
	serviceTarget       = "/lib/systemd/system/alpamon.service"
)

type ConfigData struct {
	URL    string
	ID     string
	Key    string
	Verify string
	CACert string
	Debug  string
}

//go:embed configs/*
var configFiles embed.FS

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Alpamon agent as a service",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Installing systemd service for alpamon...")

		err := copyEmbeddedFile(tmpFilePath, tmpFileTarget)
		if err != nil {
			return err
		}

		output, err := exec.Command("systemd-tmpfiles", "--create").CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w\n%s", err, string(output))
		}

		err = writeConfig()
		if err != nil {
			return err
		}

		err = writeService()
		if err != nil {
			return err
		}

		return nil
	},
}

func writeConfig() error {
	if isConfigValid(configTarget) {
		return nil
	}

	tmplData, err := configFiles.ReadFile(configTemplatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file (%s): %v", configTemplatePath, err)
	}

	tmpl, err := template.New("alpamon.conf").Parse(string(tmplData))
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	configData := ConfigData{
		URL:    utils.GetEnvOrDefault("ALPACON_URL", ""),
		ID:     utils.GetEnvOrDefault("PLUGIN_ID", ""),
		Key:    utils.GetEnvOrDefault("PLUGIN_KEY", ""),
		Verify: utils.GetEnvOrDefault("ALPACON_SSL_VERIFY", "true"),
		CACert: utils.GetEnvOrDefault("ALPACON_CA_CERT", ""),
		Debug:  utils.GetEnvOrDefault("PLUGIN_DEBUG", "true"),
	}

	if configData.URL == "" || configData.ID == "" || configData.Key == "" {
		return fmt.Errorf("environment variables ALPACON_URL, PLUGIN_ID, PLUGIN_KEY must be set")
	}

	tmpFile, err := os.CreateTemp("", "alpamon.conf")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer func() { _ = tmpFile.Close() }()

	err = tmpl.Execute(tmpFile, configData)
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		return fmt.Errorf("failed to execute template: %v", err)
	}

	err = os.MkdirAll(filepath.Dir(configTarget), 0755)
	if err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	err = os.Rename(tmpFile.Name(), configTarget)
	if err != nil {
		return fmt.Errorf("failed to move temp file to target: %v", err)
	}

	return nil
}

func writeService() error {
	if isConfigValid(serviceTarget) {
		return nil
	}

	err := copyEmbeddedFile(serviceTemplatePath, serviceTarget)
	if err != nil {
		return fmt.Errorf("failed to write target file: %v", err)
	}

	return nil
}

func copyEmbeddedFile(srcPath, dstPath string) error {
	fileData, err := configFiles.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read embedded file: %v", err)
	}

	outFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer func() { _ = outFile.Close() }()

	_, err = outFile.Write(fileData)
	if err != nil {
		return fmt.Errorf("failed to write to destination file: %v", err)
	}

	return nil
}

func isConfigValid(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.Size() > 0
}
