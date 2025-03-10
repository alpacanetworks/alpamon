package setup

import (
	"embed"
	"fmt"
	cli "github.com/alpacanetworks/alpacon-cli/utils"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"text/template"
)

//go:embed configs/*
var configFiles embed.FS

var (
	name                string
	configTemplatePath  string
	configTarget        string
	tmpFilePath         = "configs/tmpfile.conf"
	tmpFileTarget       string
	serviceTemplatePath string
	serviceTarget       string
)

func SetConfigPaths(serviceName string) {
	name = serviceName
	configTemplatePath = fmt.Sprintf("configs/%s.conf", name)
	configTarget = fmt.Sprintf("/etc/%s/%s.conf", name, name)
	tmpFileTarget = fmt.Sprintf("/usr/lib/tmpfiles.d/%s.conf", name)
	serviceTemplatePath = fmt.Sprintf("configs/%s.service", name)
	serviceTarget = fmt.Sprintf("/lib/systemd/system/%s.service", name)
}

type ConfigData struct {
	URL    string
	ID     string
	Key    string
	Verify string
	CACert string
	Debug  string
}

var SetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup and configure the alpamon",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Starting %s setup...\n", name)

		configExists := fileExists(configTarget)
		isOverwrite := true

		if term.IsTerminal(syscall.Stdin) {
			if configExists {
				fmt.Println("A configuration file already exists at:", configTarget)
				isOverwrite = cli.PromptForBool("Do you want to overwrite it with a new configuration?: ")
			}

			if !isOverwrite {
				fmt.Println("Keeping the existing configuration file. Skipping configuration update.")
				return nil
			}
		}

		if !configExists || isOverwrite {
			fmt.Println("Applying a new configuration automatically.")
		}

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

		fmt.Println("Configuration file successfully updated.")
		return nil
	},
}

func writeConfig() error {
	tmplData, err := configFiles.ReadFile(configTemplatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file (%s): %v", configTemplatePath, err)
	}

	tmpl, err := template.New(fmt.Sprintf("%s.conf", name)).Parse(string(tmplData))
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

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("%s.conf", name))
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

func fileExists(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.Size() > 0
}
