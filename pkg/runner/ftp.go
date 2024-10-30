package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/alpacanetworks/alpamon-go/pkg/config"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type FtpClient struct {
	conn             *websocket.Conn
	requestHeader    http.Header
	url              string
	username         string
	groupname        string
	homeDirectory    string
	workingDirectory string
}

func NewFtpClient(data CommandData) *FtpClient {
	headers := http.Header{
		"Authorization": {fmt.Sprintf(`id="%s", key="%s"`, config.FtpSettings.ID, config.FtpSettings.Key)},
		"Origin":        {config.FtpSettings.ServerURL},
	}

	return &FtpClient{
		requestHeader:    headers,
		url:              strings.Replace(config.FtpSettings.ServerURL, "http", "ws", 1) + data.URL,
		username:         data.Username,
		groupname:        data.Groupname,
		homeDirectory:    data.HomeDirectory,
		workingDirectory: data.HomeDirectory,
	}
}

func (fc *FtpClient) demote() error {
	if syscall.Getuid() == 0 {
		if fc.username == "" || fc.groupname == "" {
			log.Debug().Msg("No username or groupname provided.")
			return fmt.Errorf("no username or groupname provided")
		}

		usr, err := user.Lookup(fc.username)
		if err != nil {
			log.Debug().Msgf("There is no corresponding %s username in this server", fc.username)
			return fmt.Errorf("there is no corresponding %s username in this server", fc.username)
		}

		group, err := user.LookupGroup(fc.groupname)
		if err != nil {
			log.Debug().Msgf("There is no corresponding %s groupname in this server", fc.groupname)
			return fmt.Errorf("there is no corresponding %s groupname in this server", fc.groupname)
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

	return nil
}

func (fc *FtpClient) RunFtpBackground() {
	log.Debug().Msg("Opening websocket for ftp session.")

	var err error
	err = fc.demote()
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get demote permission")
		return
	}

	fc.conn, _, err = websocket.DefaultDialer.Dial(fc.url, fc.requestHeader)
	if err != nil {
		log.Debug().Err(err).Msgf("Failed to connect to pty websocket at %s", fc.url)
		return
	}
	defer fc.close()

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
				log.Debug().Err(err).Msg("Failed to read from ftp websocket")
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

	entries, err := os.ReadDir(path)
	if err != nil {
		return CommandResult{
			Name:    filepath.Base(path),
			Path:    path,
			Message: err.Error(),
		}, nil
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		child := CommandResult{
			Name: entry.Name(),
			Path: fullPath,
			Size: info.Size(),
		}

		if entry.IsDir() {
			child.Type = "folder"
			if current < depth-1 {
				childResult, err := fc.listRecursive(fullPath, depth, current+1)
				if err != nil {
					continue
				}
				child.Children = childResult.Children
				child.Size = childResult.Size
			}
		} else {
			child.Type = "file"
		}

		result.Children = append(result.Children, child)
		result.Size += child.Size
	}

	return result, nil
}

func (fc *FtpClient) mkd(path string) (CommandResult, error) {
	path = fc.parsePath(path)

	err := os.Mkdir(path, 0755)
	if err != nil {
		return CommandResult{
			Message: err.Error(),
		}, err
	}

	return CommandResult{
		Message: fmt.Sprintf("Make %s successfully", path),
	}, nil
}

func (fc *FtpClient) cwd(path string) (CommandResult, error) {
	path = fc.parsePath(path)

	info, err := os.Stat(path)
	if err != nil {
		return CommandResult{
			Message: err.Error(),
		}, err
	}

	if !info.IsDir() {
		return CommandResult{
			Message: "not a directory",
		}, fmt.Errorf("not a directory")
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

	err := os.Remove(path)
	if err != nil {
		return CommandResult{
			Message: err.Error(),
		}, err
	}

	return CommandResult{
		Message: fmt.Sprintf("Delete %s successfully", path),
	}, nil
}

func (fc *FtpClient) rmd(path string, recursive bool) (CommandResult, error) {
	path = fc.parsePath(path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return CommandResult{
			Message: err.Error(),
		}, err
	}

	var err error
	if recursive {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}

	if err != nil {
		return CommandResult{
			Message: err.Error(),
		}, err
	}

	return CommandResult{
		Message: fmt.Sprintf("Delete %s successfully", path),
	}, nil
}

func (fc *FtpClient) mv(src, dst string) (CommandResult, error) {
	src = fc.parsePath(src)
	dst = filepath.Join(fc.parsePath(dst), filepath.Base(src))

	err := os.Rename(src, dst)
	if err != nil {
		return CommandResult{
			Message: err.Error(),
		}, err
	}

	return CommandResult{
		Message: fmt.Sprintf("Move %s to %s", src, dst),
	}, nil
}

func (fc *FtpClient) cp(src, dst string) (CommandResult, error) {
	src = fc.parsePath(src)
	dst = filepath.Join(fc.parsePath(dst), filepath.Base(src))

	info, err := os.Stat(src)
	if err != nil {
		return CommandResult{
			Message: err.Error(),
		}, err
	}

	if info.IsDir() {
		return fc.cpDir(src, dst)
	}
	return fc.cpFile(src, dst)
}

func (fc *FtpClient) cpDir(src, dst string) (CommandResult, error) {
	err := copyDir(src, dst)
	if err != nil {
		return CommandResult{
			Message: err.Error(),
		}, err
	}

	return CommandResult{
		Message: fmt.Sprintf("Copy %s to %s", src, dst),
	}, nil
}

func (fc *FtpClient) cpFile(src, dst string) (CommandResult, error) {
	err := copyFile(src, dst)
	if err != nil {
		return CommandResult{
			Message: err.Error(),
		}, err
	}

	return CommandResult{
		Message: fmt.Sprintf("Copy %s to %s", src, dst),
	}, nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	if err = dstFile.Close(); err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err = os.Chmod(dst, srcInfo.Mode()); err != nil {
		return err
	}

	return nil
}

func copyDir(src, dst string) error {
	if strings.HasPrefix(dst, src) {
		return fmt.Errorf("%s is inside %s, causing infinite recursion", dst, src)
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err = copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err = copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
