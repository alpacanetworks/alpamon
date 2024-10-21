package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/alpacanetworks/alpamon-go/pkg/config"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type FtpClient struct {
	conn             *websocket.Conn
	requestHeader    http.Header
	sysProcAttr      *syscall.SysProcAttr
	url              string
	username         string
	groupname        string
	homeDirectory    string
	workingDirectory string
	sessionID        string
}

var ftpTerminals map[string]*FtpClient

func init() {
	ftpTerminals = make(map[string]*FtpClient)
}

func NewFtpClient(data CommandData) *FtpClient {
	headers := http.Header{
		"Authorization": {fmt.Sprintf(`id="%s", key="%s"`, config.GlobalSettings.ID, config.GlobalSettings.Key)},
		"Origin":        {config.GlobalSettings.ServerURL},
	}

	return &FtpClient{
		requestHeader:    headers,
		url:              strings.Replace(config.GlobalSettings.ServerURL, "http", "ws", 1) + data.URL,
		username:         data.Username,
		groupname:        data.Groupname,
		homeDirectory:    data.HomeDirectory,
		workingDirectory: data.HomeDirectory,
		sessionID:        data.SessionID,
	}
}

func (fc *FtpClient) RunFtpBackground() {
	log.Debug().Msg("Opening websocket for ftp session.")

	var err error
	fc.conn, _, err = websocket.DefaultDialer.Dial(fc.url, fc.requestHeader)
	if err != nil {
		log.Debug().Err(err).Msgf("Failed to connect to pty websocket at %s", fc.url)
		return
	}

	fc.sysProcAttr, err = demote(fc.username, fc.groupname)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to demote user.")
		fc.close()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go fc.read(ctx, cancel)

	ftpTerminals[fc.sessionID] = fc

	<-ctx.Done()
	fc.close()
}

func (fc *FtpClient) read(ctx context.Context, cancel context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, message, err := fc.conn.ReadMessage()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Debug().Err(err).Msg("Failed to read from ftp websocket")
				cancel()
				return
			}

			var content map[string]interface{}
			err = json.Unmarshal(message, &content)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to unmarshal websocket message")
				cancel()
				return
			}
			command := content["command"].(string)
			result := map[string]interface{}{
				"command": command,
				"success": true,
			}

			data, err := fc.handleFtpCommand(command, content["data"].(map[string]interface{}))
			if err != nil {
				result["success"] = false
				result["data"], result["code"] = GetFtpErrorCode(command, data.(map[string]string))
			} else {
				result["code"] = returnCodes[command].Success
				result["data"] = data
			}

			response, err := json.Marshal(result)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Debug().Err(err).Msg("Failed to marshal response")
				cancel()
				return
			}

			err = fc.conn.WriteMessage(websocket.TextMessage, response)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Debug().Err(err).Msg("Failed to send websocket message")
				cancel()
				return
			}
		}
	}
}

func (fc *FtpClient) close() {
	if fc.conn != nil {
		err := fc.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Debug().Err(err).Msg("Failed to write close message to ftp websocket")
		}
		_ = fc.conn.Close()
	}

	if ftpTerminals[fc.sessionID] != nil {
		delete(ftpTerminals, fc.sessionID)
	}

	log.Debug().Msg("Websocket connection for ftp has been closed.")
}

func (fc *FtpClient) handleFtpCommand(command string, data map[string]interface{}) (interface{}, error) {
	switch command {
	case "list":
		return fc.list(data["path"].(string), int(data["depth"].(float64)))
	case "mkd":
		return fc.mkd(data["path"].(string))
	case "cwd":
		return fc.cwd(data["path"].(string))
	case "pwd":
		return fc.pwd()
	case "dele":
		return fc.dele(data["path"].(string))
	case "rmd":
		return fc.rmd(data["path"].(string), data["recursive"].(bool))
	case "mv":
		return fc.mv(data["src"].(string), data["dst"].(string))
	case "cp":
		return fc.cp(data["src"].(string), data["dst"].(string))
	default:
		return nil, fmt.Errorf("unknown FTP command: %s", command)
	}
}

func (fc *FtpClient) parsePath(path string) string {
	if strings.HasPrefix(path, "~") {
		path = strings.Replace(path, "~", fc.workingDirectory, 1)
	}

	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(fc.workingDirectory, path)
	}

	parsedPath := filepath.Clean(absPath)
	return parsedPath
}

func (fc *FtpClient) size(path string) (int64, error) {
	cmd := exec.Command("du", "-sk", path)
	cmd.SysProcAttr = fc.sysProcAttr
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	parts := strings.Fields(string(output))
	if len(parts) < 1 {
		return 0, fmt.Errorf("could not retrieve size for path: %s", path)
	}

	size := int64(0)
	if _, err := fmt.Sscanf(parts[0], "%d", &size); err != nil {
		return size, err
	}

	return size * 1024, nil
}

func (fc *FtpClient) isDir(path string) bool {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("ls -ld \"%s\" | awk '{print $1}'", path))
	cmd.SysProcAttr = fc.sysProcAttr
	output, _ := cmd.Output()

	return strings.HasPrefix(string(output), "d")
}

func (fc *FtpClient) list(rootDir string, depth int) (map[string]interface{}, error) {
	path := fc.parsePath(rootDir)
	return fc.listRecursive(path, depth, 0)
}

func (fc *FtpClient) listRecursive(path string, depth, current int) (map[string]interface{}, error) {

	result := map[string]interface{}{
		"name":     filepath.Base(path),
		"type":     "folder",
		"path":     path,
		"size":     int64(0),
		"children": []interface{}{},
	}

	cmd := exec.Command("find", path, "-mindepth", "1", "-maxdepth", "1")
	cmd.SysProcAttr = fc.sysProcAttr
	output, err := cmd.CombinedOutput()
	if err != nil {
		result = map[string]interface{}{
			"name":    filepath.Base(path),
			"path":    path,
			"message": string(output),
		}
		return result, nil
	}

	paths := strings.Split(string(output), "\n")
	for _, foundPath := range paths {
		if foundPath == "" {
			continue
		}

		size, err := fc.size(foundPath)
		if err != nil {
			result = map[string]interface{}{
				"name":    filepath.Base(path),
				"path":    path,
				"message": string(output),
			}
			return result, nil
		}

		child := map[string]interface{}{
			"name": filepath.Base(foundPath),
			"path": foundPath,
			"size": size,
		}

		if fc.isDir(foundPath) {
			child["type"] = "folder"

			if current < depth-1 {
				childResult, err := fc.listRecursive(foundPath, depth, current+1)
				if err != nil {
					return result, nil
				}
				child["children"] = childResult["children"]
				child["size"] = childResult["size"]
			} else {
				child["children"] = []interface{}{}
			}
		} else {
			child["type"] = "file"
		}

		result["children"] = append(result["children"].([]interface{}), child)
		result["size"] = result["size"].(int64) + size
	}

	return result, nil
}

func (fc *FtpClient) mkd(path string) (map[string]string, error) {
	path = fc.parsePath(path)

	cmd := exec.Command("mkdir", path)
	cmd.SysProcAttr = fc.sysProcAttr
	output, err := cmd.CombinedOutput()
	if err != nil {
		return map[string]string{
			"message": strings.ToLower(string(output)),
		}, err
	}

	return map[string]string{
		"message": fmt.Sprintf("Make %s successfully", path),
	}, nil
}

func (fc *FtpClient) cwd(path string) (map[string]string, error) {
	path = fc.parsePath(path)
	cmd := exec.Command("test", "-r", path, "-a", "-w", path, "-a", "-x", path)
	cmd.SysProcAttr = fc.sysProcAttr
	output, err := cmd.CombinedOutput()
	if err != nil {
		return map[string]string{
			"message": strings.ToLower(string(output)),
		}, err
	}
	fc.workingDirectory = path

	return map[string]string{
		"message": fmt.Sprintf("Change working directory to %s", path),
	}, nil
}

func (fc *FtpClient) pwd() (map[string]string, error) {
	return map[string]string{
		"message": fmt.Sprintf("Current working directory: %s", fc.workingDirectory),
		"path":    fc.workingDirectory,
	}, nil
}

func (fc *FtpClient) dele(path string) (map[string]string, error) {
	path = fc.parsePath(path)

	cmd := exec.Command("rm", path)
	cmd.SysProcAttr = fc.sysProcAttr
	if output, err := cmd.CombinedOutput(); err != nil {
		return map[string]string{
			"message": strings.ToLower(string(output)),
		}, err
	}

	return map[string]string{
		"message": fmt.Sprintf("Delete %s successfully", path),
	}, nil
}

func (fc *FtpClient) rmd(path string, recursive bool) (map[string]string, error) {
	path = fc.parsePath(path)

	var cmd *exec.Cmd
	if recursive {
		cmd = exec.Command("rm", "-r", path)
	} else {
		cmd = exec.Command("rmdir", path)
	}

	cmd.SysProcAttr = fc.sysProcAttr
	if output, err := cmd.CombinedOutput(); err != nil {
		return map[string]string{
			"message": strings.ToLower(string(output)),
		}, err
	}

	return map[string]string{
		"message": fmt.Sprintf("Delete %s successfully", path),
	}, nil
}

func (fc *FtpClient) mv(src, dst string) (map[string]string, error) {
	src = fc.parsePath(src)
	dst = filepath.Join(fc.parsePath(dst), filepath.Base(src))
	cmd := exec.Command("mv", src, dst)
	cmd.SysProcAttr = fc.sysProcAttr
	if output, err := cmd.CombinedOutput(); err != nil {
		return map[string]string{
			"message": strings.ToLower(string(output)),
		}, err
	}

	return map[string]string{
		"message": fmt.Sprintf("Move %s to %s", src, dst),
	}, nil
}

func (fc *FtpClient) cp(src, dst string) (map[string]string, error) {
	src = fc.parsePath(src)
	dst = filepath.Join(fc.parsePath(dst), filepath.Base(src))

	if fc.isDir(src) {
		return fc.cpDir(src, dst)
	}
	return fc.cpFile(src, dst)
}

func (fc *FtpClient) cpDir(src, dst string) (map[string]string, error) {
	cmd := exec.Command("cp", "-r", src+"/*", dst)
	cmd.SysProcAttr = fc.sysProcAttr
	if output, err := cmd.CombinedOutput(); err != nil {
		return map[string]string{
			"message": strings.ToLower(string(output)),
		}, err
	}

	return map[string]string{
		"message": fmt.Sprintf("Copy %s to %s", src, dst),
	}, nil
}

func (fc *FtpClient) cpFile(src, dst string) (map[string]string, error) {
	cmd := exec.Command("cp", src, dst)
	cmd.SysProcAttr = fc.sysProcAttr
	if output, err := cmd.CombinedOutput(); err != nil {
		return map[string]string{
			"message": strings.ToLower(string(output)),
		}, err
	}

	return map[string]string{
		"message": fmt.Sprintf("Copy %s to %s", src, dst),
	}, nil
}
