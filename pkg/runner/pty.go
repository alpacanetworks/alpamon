package runner

import (
	"context"
	"errors"
	"fmt"
	"github.com/alpacanetworks/alpamon-go/pkg/config"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type PtyClient struct {
	conn          *websocket.Conn
	requestHeader http.Header
	cmd           *exec.Cmd
	ptmx          *os.File
	url           string
	rows          uint16
	cols          uint16
	username      string
	groupname     string
	homeDirectory string
	sessionID     string
}

var terminals map[string]*PtyClient

func init() {
	terminals = make(map[string]*PtyClient)
}

func NewPtyClient(data CommandData) *PtyClient {
	headers := http.Header{
		"Authorization": {fmt.Sprintf(`id="%s", key="%s"`, config.GlobalSettings.ID, config.GlobalSettings.Key)},
		"Origin":        {config.GlobalSettings.ServerURL},
	}

	return &PtyClient{
		requestHeader: headers,
		url:           strings.Replace(config.GlobalSettings.ServerURL, "http", "ws", 1) + data.URL,
		rows:          data.Rows,
		cols:          data.Cols,
		username:      data.Username,
		groupname:     data.Groupname,
		homeDirectory: data.HomeDirectory,
		sessionID:     data.SessionID,
	}
}

func (pc *PtyClient) RunPtyBackground() {
	log.Debug().Msg("Opening websocket for pty session.")

	var err error
	pc.conn, _, err = websocket.DefaultDialer.Dial(pc.url, pc.requestHeader)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to connect to pty websocket at %s", pc.url)
		return
	}
	defer pc.close()

	pc.cmd = exec.Command("/bin/bash", "-i")

	uid, gid, groupIds, env, err := pc.getPtyUserAndEnv()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get pty user and env")
		return
	}

	pc.setPtyCmdSysProcAttrAndEnv(uid, gid, groupIds, env)

	initialSize := &pty.Winsize{
		Rows: pc.rows,
		Cols: pc.cols,
	}

	pc.ptmx, err = pty.StartWithSize(pc.cmd, initialSize)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start pty")
		pc.close()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pc.readFromWebsocket(ctx, cancel)
	pc.readFromPTY(ctx, cancel)

	terminals[pc.sessionID] = pc

	<-ctx.Done()
}

func (pc *PtyClient) readFromWebsocket(ctx context.Context, cancel context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, message, err := pc.conn.ReadMessage()
			if err != nil {
				// Double-check ctx.Err() to handle cancellation during blocking read
				if ctx.Err() != nil {
					return
				}
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Debug().Err(err).Msg("Failed to read from pty websocket")
				}
				cancel()
				return
			}
			_, err = pc.ptmx.Write(message)
			if err != nil {
				// Double-check ctx.Err() to handle cancellation during blocking write
				if ctx.Err() != nil {
					return
				}
				if !errors.Is(err, os.ErrClosed) {
					log.Debug().Err(err).Msg("Failed to write to pty")
				}
				cancel()
				return
			}
		}
	}
}

func (pc *PtyClient) readFromPTY(ctx context.Context, cancel context.CancelFunc) {
	buf := make([]byte, 2048)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := pc.ptmx.Read(buf)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				if err == io.EOF {
					log.Debug().Msg("pty session exited.")
				} else {
					log.Debug().Err(err).Msg("Failed to read from PTY")
				}
				cancel()
				return
			}
			err = pc.conn.WriteMessage(websocket.TextMessage, buf[:n])
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Debug().Err(err).Msg("Failed to write to pty")
				}
				cancel()
				return
			}
		}
	}
}

func (pc *PtyClient) resize(rows, cols uint16) error {
	err := pty.Setsize(pc.ptmx, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to resize terminal")
		return err
	}
	pc.rows = rows
	pc.cols = cols
	log.Debug().Msgf("Resized terminal for %s to %dx%d.", pc.sessionID, pc.rows, pc.cols)
	return nil
}

// close terminates the PTY session and cleans up resources.
// It ensures that the PTY, command, and WebSocket connection are properly closed.
func (pc *PtyClient) close() {
	if pc.ptmx != nil {
		_ = pc.ptmx.Close()
	}

	if pc.cmd != nil && pc.cmd.Process != nil {
		_ = pc.cmd.Process.Kill()
		_ = pc.cmd.Wait()
	}

	if pc.conn != nil {
		_ = pc.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		_ = pc.conn.Close()
	}

	if terminals[pc.sessionID] != nil {
		delete(terminals, pc.sessionID)
	}

	log.Debug().Msg("Websocket connection for pty has been closed.")
}

// getPtyUserAndEnv retrieves user information and sets environment variables.
func (pc *PtyClient) getPtyUserAndEnv() (uid, gid int, groupIds []string, env map[string]string, err error) {
	env = getDefaultEnv()

	usr, err := utils.GetSystemUser(pc.username)
	if err != nil {
		return 0, 0, nil, nil, err
	}

	currentUID := os.Geteuid()
	if currentUID != 0 || pc.username == "" {
		env["USER"] = usr.Username
		env["HOME"] = usr.HomeDir
	} else {
		env["USER"] = pc.username
		env["HOME"] = pc.homeDirectory
	}

	uid, err = strconv.Atoi(usr.Uid)
	if err != nil {
		return 0, 0, nil, nil, fmt.Errorf("failed to convert UID: %w", err)
	}

	gid, err = strconv.Atoi(usr.Gid)
	if err != nil {
		return 0, 0, nil, nil, fmt.Errorf("failed to convert GID: %w", err)
	}

	groupIds, err = usr.GroupIds()
	if err != nil {
		return 0, 0, nil, nil, fmt.Errorf("failed to get group IDs: %w", err)
	}

	return uid, gid, groupIds, env, nil
}
