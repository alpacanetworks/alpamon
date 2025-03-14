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

var (
	name                 string
	configFiles          embed.FS
	configPath           string
	configTarget         string
	tmpFilePath          = "configs/tmpfile.conf"
	tmpFileTarget        string
	servicePath          string
	restartServicePath   string
	serviceTarget        string
	restartServiceTarget string
	timerPath            string
	timerTarget          string
)

type ConfigData struct {
	URL    string
	ID     string
	Key    string
	Verify string
	CACert string
	Debug  string
}

func SetConfigPaths(serviceName string, fs embed.FS) {
	name = serviceName
	configFiles = fs
	configPath = fmt.Sprintf("configs/%s.conf", name)
	configTarget = fmt.Sprintf("/etc/alpamon/%s.conf", name)
	tmpFileTarget = fmt.Sprintf("/usr/lib/tmpfiles.d/%s.conf", name)
	servicePath = fmt.Sprintf("configs/%s.service", name)
	serviceTarget = fmt.Sprintf("/lib/systemd/system/%s.service", name)
	restartServicePath = fmt.Sprintf("configs/%s-restart.service", name)
	restartServiceTarget = fmt.Sprintf("/lib/systemd/system/%s-restart.service", name)
	timerPath = fmt.Sprintf("configs/%s-restart.timer", name)
	timerTarget = fmt.Sprintf("/lib/systemd/system/%s-restart.timer", name)
}

var SetupCmd = &cobra.Command{
	Use:   "setup",
	Short: fmt.Sprintf("Setup and configure the %s", name),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Starting %s setup...\n", name)

		var isOverwrite bool
		configExists := fileExists(configTarget)

		if configExists && term.IsTerminal(syscall.Stdin) {
			fmt.Println("A configuration file already exists at:", configTarget)
			isOverwrite = cli.PromptForBool("Do you want to overwrite it with a new configuration?: ")
			if !isOverwrite {
				fmt.Println("Keeping the existing configuration file. Skipping configuration update.")
				return nil
			}
		}

		fmt.Println("Applying a new configuration automatically...")

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

		err = writeSystemdFiles()
		if err != nil {
			return err
		}

		fmt.Println("Configuration file successfully updated.")
		return nil
	},
}

func writeConfig() error {
	tmplData, err := configFiles.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read template file (%s): %v", configPath, err)
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

func writeSystemdFiles() error {
	err := copyEmbeddedFile(servicePath, serviceTarget)
	if err != nil {
		return fmt.Errorf("failed to write target file: %v", err)
	}

	err = copyEmbeddedFile(restartServicePath, restartServiceTarget)
	if err != nil {
		return fmt.Errorf("failed to write target file: %v", err)
	}

	err = copyEmbeddedFile(timerPath, timerTarget)
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
