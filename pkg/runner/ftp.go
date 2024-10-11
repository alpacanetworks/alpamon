package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alpacanetworks/alpamon-go/pkg/config"
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type FtpClient struct {
	conn          *websocket.Conn
	requestHeader http.Header
	cmd           *exec.Cmd
	ptmx          *os.File
	url           string
	username      string
	groupname     string
	homeDirectory string
	sessionID     string
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
		requestHeader: headers,
		url:           strings.Replace(config.GlobalSettings.ServerURL, "http", "ws", 1) + data.URL,
		username:      data.Username,
		groupname:     data.Groupname,
		homeDirectory: data.HomeDirectory,
		sessionID:     data.SessionID,
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

	fc.cmd = exec.Command("/bin/bash", "-i")

	uid, gid, groupIds, env, err := fc.getFtpUserAndEnv()
	if err != nil {
		log.Debug().Err(err).Msgf("Failed to get web ftp user and env")
		return
	}

	fc.setFtpCmdSysProcAttrAndEnv(uid, gid, groupIds, env)
	fc.ptmx, err = pty.Start(fc.cmd)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to start pty")
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
			if err := json.Unmarshal(message, &content); err != nil {
				if ctx.Err() != nil {
					return
				}
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
				result["code"] = GetFtpErrorCode(command, err)
				result["data"] = map[string]string{"message": err.Error()}
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
	if fc.ptmx != nil {
		_ = fc.ptmx.Close()
	}

	if fc.cmd != nil && fc.cmd.Process != nil {
		_ = fc.cmd.Process.Kill()
		_ = fc.cmd.Wait()
	}

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

func (fc *FtpClient) getFtpUserAndEnv() (uid, gid int, groupIds []string, env map[string]string, err error) {
	var usr *user.User
	env = getDefaultEnv()

	currentUID := os.Geteuid()
	if currentUID != 0 || fc.username == "" {
		usr, err = user.Current()
		if err != nil {
			return 0, 0, nil, env, fmt.Errorf("failed to get current user: %w", err)
		}
	} else {
		usr, err = user.Lookup(fc.username)
		if err != nil {
			return 0, 0, nil, env, fmt.Errorf("failed to lookup specified user: %w", err)
		}
	}

	env["USER"] = usr.Username
	env["HOME"] = usr.HomeDir

	uid, err = strconv.Atoi(usr.Uid)
	if err != nil {
		return 0, 0, nil, env, fmt.Errorf("failed to convert UID: %w", err)
	}

	gid, err = strconv.Atoi(usr.Gid)
	if err != nil {
		return 0, 0, nil, env, fmt.Errorf("failed to convert GID: %w", err)
	}

	groupIds, err = usr.GroupIds()
	if err != nil {
		return 0, 0, nil, env, fmt.Errorf("failed to get group IDs: %w", err)
	}

	return uid, gid, groupIds, env, nil
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
		path = strings.Replace(path, "~", fc.homeDirectory, 1)
	}

	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(fc.homeDirectory, path)
	}

	cleanPath := filepath.Clean(absPath)
	return cleanPath
}

func (fc *FtpClient) list(rootdir string, depth int) (map[string]interface{}, error) {
	path := fc.parsePath(rootdir)
	return fc.listRecursive(path, depth, 0)
}

func (fc *FtpClient) listRecursive(path string, depth, current int) (map[string]interface{}, error) {
	info, err := os.Stat(path)
	if err != nil {
		return map[string]interface{}{
			"name":    filepath.Base(path),
			"path":    path,
			"message": err.Error(),
		}, nil
	}

	result := map[string]interface{}{
		"name": info.Name(),
		"type": "file",
		"path": path,
		"size": info.Size(),
	}

	if info.IsDir() {
		result["type"] = "folder"
		result["children"] = []interface{}{}

		if current < depth {
			files, err := os.ReadDir(path)
			if err != nil {
				return result, nil
			}

			for _, file := range files {
				child, _ := fc.listRecursive(filepath.Join(path, file.Name()), depth, current+1)
				result["children"] = append(result["children"].([]interface{}), child)
				result["size"] = result["size"].(int64) + child["size"].(int64)
			}
		}
	}

	return result, nil
}

func (fc *FtpClient) mkd(path string) (map[string]string, error) {
	path = fc.parsePath(path)
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}
	return map[string]string{"message": fmt.Sprintf("Make %s successfully", path)}, nil
}

func (fc *FtpClient) cwd(path string) (map[string]string, error) {
	path = fc.parsePath(path)
	if err := os.Chdir(path); err != nil {
		return nil, err
	}
	return map[string]string{"message": fmt.Sprintf("Change working directory to %s", path)}, nil
}

func (fc *FtpClient) pwd() (map[string]string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"message": fmt.Sprintf("Current working directory: %s", dir),
		"path":    dir,
	}, nil
}

func (fc *FtpClient) dele(path string) (map[string]string, error) {
	path = fc.parsePath(path)
	if err := os.Remove(path); err != nil {
		return nil, err
	}
	return map[string]string{"message": fmt.Sprintf("Delete %s successfully", path)}, nil
}

func (fc *FtpClient) rmd(path string, recursive bool) (map[string]string, error) {
	path = fc.parsePath(path)
	var err error
	if recursive {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}
	if err != nil {
		return nil, err
	}
	return map[string]string{"message": fmt.Sprintf("Delete %s successfully", path)}, nil
}

func (fc *FtpClient) mv(src, dst string) (map[string]string, error) {
	src = fc.parsePath(src)
	dst = filepath.Join(fc.parsePath(dst), filepath.Base(src))
	if err := os.Rename(src, dst); err != nil {
		return nil, err
	}
	return map[string]string{"message": fmt.Sprintf("Move %s to %s", src, dst)}, nil
}

func (fc *FtpClient) cp(src, dst string) (map[string]string, error) {
	src = fc.parsePath(src)
	dst = filepath.Join(fc.parsePath(dst), filepath.Base(src))

	srcInfo, err := os.Stat(src)
	if err != nil {
		return nil, err
	}

	if srcInfo.IsDir() {
		return fc.cpDir(src, dst)
	}
	return fc.cpFile(src, dst)
}

func (fc *FtpClient) cpDir(src, dst string) (map[string]string, error) {
	if err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		_, err = fc.cpFile(path, dstPath)
		return err
	}); err != nil {
		return nil, err
	}

	return map[string]string{"message": fmt.Sprintf("Copy %s to %s", src, dst)}, nil
}

func (fc *FtpClient) cpFile(src, dst string) (map[string]string, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return nil, err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return nil, err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return nil, err
	}

	return map[string]string{"message": fmt.Sprintf("Copy %s to %s", src, dst)}, os.Chmod(dst, srcInfo.Mode())
}
