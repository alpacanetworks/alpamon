package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/alpacanetworks/alpamon-go/pkg/config"
	"github.com/alpacanetworks/alpamon-go/pkg/logger"
	"github.com/gorilla/websocket"
)

type FtpClient struct {
	conn             *websocket.Conn
	requestHeader    http.Header
	url              string
	homeDirectory    string
	workingDirectory string
	log              logger.FtpLogger
	settings         config.Settings
}

func NewFtpClient(data FtpConfigData) *FtpClient {
	headers := http.Header{
		"Authorization": {fmt.Sprintf(`id="%s", key="%s"`, data.Settings.ID, data.Settings.Key)},
		"Origin":        {data.Settings.ServerURL},
	}

	return &FtpClient{
		requestHeader:    headers,
		url:              strings.Replace(data.Settings.ServerURL, "http", "ws", 1) + data.URL,
		homeDirectory:    data.HomeDirectory,
		workingDirectory: data.HomeDirectory,
		log:              data.Logger,
		settings:         data.Settings,
	}
}

func (fc *FtpClient) RunFtpBackground() {
	fc.log.Debug().Msg("Opening websocket for ftp session.")

	var err error
	fc.conn, _, err = websocket.DefaultDialer.Dial(fc.url, fc.requestHeader)
	if err != nil {
		fc.log.Debug().Err(err).Msgf("Failed to connect to pty websocket at %s", fc.url)
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
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					fc.log.Debug().Err(err).Msg("Failed to read from ftp websocket")
				}
				cancel()
				return
			}

			var content FtpContent
			err = json.Unmarshal(message, &content)
			if err != nil {
				fc.log.Debug().Err(err).Msg("Failed to unmarshal websocket message")
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
				fc.log.Debug().Err(err).Msg("Failed to marshal response")
				cancel()
				return
			}

			err = fc.conn.WriteMessage(websocket.TextMessage, response)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					fc.log.Debug().Err(err).Msg("Failed to send websocket message")
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

	fc.log.Debug().Msg("Websocket connection for ftp has been closed.")
	os.Exit(1)
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
		ModTime:  nil,
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

		modTime := info.ModTime()
		child := CommandResult{
			Name:    entry.Name(),
			Path:    fullPath,
			Size:    info.Size(),
			ModTime: &modTime,
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

	dirInfo, err := os.Stat(path)
	if err != nil {
		result.Message = err.Error()
	} else {
		modTime := dirInfo.ModTime()
		result.ModTime = &modTime
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

func copyFile(src, dst string) (finalErr error) {
	finalErr = nil
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}

	defer func() {
		if err = srcFile.Close(); err != nil && finalErr == nil {
			finalErr = err
		}
	}()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}

	defer func() {
		if err := dstFile.Close(); err != nil && finalErr == nil {
			finalErr = err
		}
	}()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err = os.Chmod(dst, srcInfo.Mode()); err != nil {
		return err
	}

	return finalErr
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
