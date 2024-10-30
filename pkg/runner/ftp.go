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
	defer fc.close()

	fc.sysProcAttr, err = demote(fc.username, fc.groupname)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to demote user.")
		fc.close()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go fc.read(ctx, cancel)

	<-ctx.Done()
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
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Debug().Err(err).Msg("Failed to read from ftp websocket")
				}
				cancel()
				return
			}

			var content FtpContent
			err = json.Unmarshal(message, &content)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to unmarshal websocket message")
				cancel()
				return
			}

			result := FtpResult{
				Command: content.Command,
				Success: true,
			}

			data, err := fc.handleFtpCommand(content.Command, content.Data)
			if err != nil {
				result.Success = false
				result.Data, result.Code = GetFtpErrorCode(content.Command, data)
			} else {
				result.Code = returnCodes[content.Command].Success
				result.Data = data
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
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Debug().Err(err).Msg("Failed to send websocket message")
				}
				cancel()
				return
			}
		}
	}
}

func (fc *FtpClient) close() {
	if fc.conn != nil {
		_ = fc.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		_ = fc.conn.Close()
	}

	log.Debug().Msg("Websocket connection for ftp has been closed.")
}

func (fc *FtpClient) handleFtpCommand(command FtpCommand, data FtpData) (CommandResult, error) {
	switch command {
	case List:
		return fc.list(data.Path, data.Depth)
	case Mkd:
		return fc.mkd(data.Path)
	case Cwd:
		return fc.cwd(data.Path)
	case Pwd:
		return fc.pwd()
	case Dele:
		return fc.dele(data.Path)
	case Rmd:
		return fc.rmd(data.Path, data.Recursive)
	case Mv:
		return fc.mv(data.Src, data.Dst)
	case Cp:
		return fc.cp(data.Src, data.Dst)
	default:
		return CommandResult{}, fmt.Errorf("unknown FTP command: %s", command)
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
	if _, err = fmt.Sscanf(parts[0], "%d", &size); err != nil {
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

func (fc *FtpClient) list(rootDir string, depth int) (CommandResult, error) {
	path := fc.parsePath(rootDir)
	cmdResult, err := fc.listRecursive(path, depth, 0)
	return cmdResult, err
}

func (fc *FtpClient) listRecursive(path string, depth, current int) (CommandResult, error) {
	result := CommandResult{
		Name:     filepath.Base(path),
		Type:     "folder",
		Path:     path,
		Size:     int64(0),
		Children: []CommandResult{},
	}

	cmd := exec.Command("find", path, "-mindepth", "1", "-maxdepth", "1")
	cmd.SysProcAttr = fc.sysProcAttr
	output, err := cmd.CombinedOutput()
	if err != nil {
		return CommandResult{
			Name:    filepath.Base(path),
			Path:    path,
			Message: string(output),
		}, nil
	}

	paths := strings.Split(string(output), "\n")
	for _, foundPath := range paths {
		if foundPath == "" {
			continue
		}

		size, err := fc.size(foundPath)
		if err != nil {
			return CommandResult{
				Name:    filepath.Base(path),
				Path:    path,
				Message: string(output),
			}, nil
		}

		child := CommandResult{
			Name: filepath.Base(foundPath),
			Path: foundPath,
			Size: size,
		}

		if fc.isDir(foundPath) {
			child.Type = "folder"

			if current < depth-1 {
				childResult, err := fc.listRecursive(foundPath, depth, current+1)
				if err != nil {
					return result, nil
				}
				child.Children = childResult.Children
				child.Size = childResult.Size
			}
		} else {
			child.Type = "file"
		}

		result.Children = append(result.Children, child)
		result.Size += size
	}

	return result, nil
}

func (fc *FtpClient) mkd(path string) (CommandResult, error) {
	path = fc.parsePath(path)

	cmd := exec.Command("mkdir", path)
	cmd.SysProcAttr = fc.sysProcAttr
	output, err := cmd.CombinedOutput()
	if err != nil {
		return CommandResult{
			Message: strings.ToLower(string(output)),
		}, err
	}

	return CommandResult{
		Message: fmt.Sprintf("Make %s successfully", path),
	}, nil
}

func (fc *FtpClient) cwd(path string) (CommandResult, error) {
	path = fc.parsePath(path)
	cmd := exec.Command("test", "-r", path, "-a", "-w", path, "-a", "-x", path)
	cmd.SysProcAttr = fc.sysProcAttr
	output, err := cmd.CombinedOutput()
	if err != nil {
		return CommandResult{
			Message: strings.ToLower(string(output)),
		}, err
	}

	fc.workingDirectory = path

	return CommandResult{
		Message: fmt.Sprintf("Change working directory to %s", path),
	}, nil
}

func (fc *FtpClient) pwd() (CommandResult, error) {
	return CommandResult{
		Message: fmt.Sprintf("Current working directory: %s", fc.workingDirectory),
		Path:    fc.workingDirectory,
	}, nil
}

func (fc *FtpClient) dele(path string) (CommandResult, error) {
	path = fc.parsePath(path)

	cmd := exec.Command("rm", path)
	cmd.SysProcAttr = fc.sysProcAttr
	if output, err := cmd.CombinedOutput(); err != nil {
		return CommandResult{
			Message: strings.ToLower(string(output)),
		}, err
	}

	return CommandResult{
		Message: fmt.Sprintf("Delete %s successfully", path),
	}, nil
}

func (fc *FtpClient) rmd(path string, recursive bool) (CommandResult, error) {
	path = fc.parsePath(path)

	var cmd *exec.Cmd
	if recursive {
		cmd = exec.Command("rm", "-r", path)
	} else {
		cmd = exec.Command("rmdir", path)
	}

	cmd.SysProcAttr = fc.sysProcAttr
	if output, err := cmd.CombinedOutput(); err != nil {
		return CommandResult{
			Message: strings.ToLower(string(output)),
		}, err
	}

	return CommandResult{
		Message: fmt.Sprintf("Delete %s successfully", path),
	}, nil
}

func (fc *FtpClient) mv(src, dst string) (CommandResult, error) {
	src = fc.parsePath(src)
	dst = filepath.Join(fc.parsePath(dst), filepath.Base(src))

	cmd := exec.Command("mv", src, dst)
	cmd.SysProcAttr = fc.sysProcAttr
	if output, err := cmd.CombinedOutput(); err != nil {
		return CommandResult{
			Message: strings.ToLower(string(output)),
		}, err
	}

	return CommandResult{
		Message: fmt.Sprintf("Move %s to %s", src, dst),
	}, nil
}

func (fc *FtpClient) cp(src, dst string) (CommandResult, error) {
	src = fc.parsePath(src)
	dst = filepath.Join(fc.parsePath(dst), filepath.Base(src))

	if fc.isDir(src) {
		return fc.cpDir(src, dst)
	}
	return fc.cpFile(src, dst)
}

func (fc *FtpClient) cpDir(src, dst string) (CommandResult, error) {
	cmd := exec.Command("cp", "-r", src, dst)
	cmd.SysProcAttr = fc.sysProcAttr
	if output, err := cmd.CombinedOutput(); err != nil {
		return CommandResult{
			Message: strings.ToLower(string(output)),
		}, err
	}

	return CommandResult{
		Message: fmt.Sprintf("Copy %s to %s", src, dst),
	}, nil
}

func (fc *FtpClient) cpFile(src, dst string) (CommandResult, error) {
	cmd := exec.Command("cp", src, dst)
	cmd.SysProcAttr = fc.sysProcAttr
	if output, err := cmd.CombinedOutput(); err != nil {
		return CommandResult{
			Message: strings.ToLower(string(output)),
		}, err
	}

	return CommandResult{
		Message: fmt.Sprintf("Copy %s to %s", src, dst),
	}, nil
}
