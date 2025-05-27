package runner

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alpacanetworks/alpamon/pkg/config"
	"github.com/alpacanetworks/alpamon/pkg/scheduler"
	"github.com/alpacanetworks/alpamon/pkg/utils"
	"github.com/cenkalti/backoff"
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type PtyClient struct {
	conn          *websocket.Conn
	apiSession    *scheduler.Session
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
	isRecovering  atomic.Bool // default : false
}

const (
	maxRecoveryTimeout     = 1 * time.Minute
	reissuePtyWebsocketURL = "/api/websh/pty-channels/"
)

var terminals map[string]*PtyClient

func init() {
	terminals = make(map[string]*PtyClient)
}

func NewPtyClient(data CommandData, apiSession *scheduler.Session) *PtyClient {
	headers := http.Header{
		"Authorization": {fmt.Sprintf(`id="%s", key="%s"`, config.GlobalSettings.ID, config.GlobalSettings.Key)},
		"Origin":        {config.GlobalSettings.ServerURL},
	}

	return &PtyClient{
		apiSession:    apiSession,
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

func (pc *PtyClient) initializePtySession() error {
	var err error
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !config.GlobalSettings.SSLVerify,
		},
	}
	pc.conn, _, err = dialer.Dial(pc.url, pc.requestHeader)
	if err != nil {
		return fmt.Errorf("failed to connect pty websocket: %w", err)
	}

	pc.cmd = exec.Command("/bin/bash", "-i")
	uid, gid, groupIds, env, err := pc.getPtyUserAndEnv()
	if err != nil {
		return fmt.Errorf("failed to get user/env: %w", err)
	}
	pc.setPtyCmdSysProcAttrAndEnv(uid, gid, groupIds, env)

	initialSize := &pty.Winsize{Rows: pc.rows, Cols: pc.cols}
	pc.ptmx, err = pty.StartWithSize(pc.cmd, initialSize)
	if err != nil {
		return fmt.Errorf("failed to start pty: %w", err)
	}

	terminals[pc.sessionID] = pc
	return nil
}

func (pc *PtyClient) RunPtyBackground() {
	log.Debug().Msg("Opening websocket for pty session.")
	defer pc.close()

	err := pc.initializePtySession()
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recoveryChan := make(chan struct{}, 1)
	recoveredWsChan := make(chan struct{}, 1)
	recoveredPtyChan := make(chan struct{}, 1)

	go pc.readFromWebsocket(ctx, cancel, recoveryChan, recoveredWsChan)
	go pc.readFromPTY(ctx, cancel, recoveryChan, recoveredPtyChan)

	for {
		select {
		case <-ctx.Done():
			return
		case <-recoveryChan:
			log.Debug().Msg("Attempting to reconnect pty websocket...")
			err = pc.recovery()
			pc.isRecovering.Store(false)
			if err != nil {
				cancel()
				return
			}
			log.Debug().Msg("Pty websocket reconnected successfully.")
			recoveredWsChan <- struct{}{}
			recoveredPtyChan <- struct{}{}
		}
	}
}

func (pc *PtyClient) readFromWebsocket(ctx context.Context, cancel context.CancelFunc, recoveryChan, recoveredChan chan struct{}) {
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

				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Debug().Msg("Pty websocket connection closed by peer.")
					cancel()
					return
				}

				if pc.isRecovering.CompareAndSwap(false, true) {
					recoveryChan <- struct{}{}
				}
				select {
				case <-recoveredChan:
					continue
				case <-ctx.Done():
					return
				}
			}
			_, err = pc.ptmx.Write(message)
			if err != nil {
				// Double-check ctx.Err() to handle cancellation during blocking write
				if ctx.Err() != nil {
					return
				}
				if !errors.Is(err, os.ErrClosed) {
					log.Debug().Err(err).Msg("Failed to write to pty.")
				}
				cancel()
				return
			}
		}
	}
}

func (pc *PtyClient) readFromPTY(ctx context.Context, cancel context.CancelFunc, recoveryChan, recoveredChan chan struct{}) {
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
					log.Debug().Err(err).Msg("Failed to read from pty.")
				}
				cancel()
				return
			}
			err = pc.conn.WriteMessage(websocket.TextMessage, buf[:n])
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Debug().Msg("Pty websocket connection closed by peer.")
					cancel()
					return
				}

				if pc.isRecovering.CompareAndSwap(false, true) {
					recoveryChan <- struct{}{}
				}
				select {
				case <-recoveredChan:
					continue
				case <-ctx.Done():
					return
				}
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
		log.Warn().Err(err).Msg("Failed to resize terminal.")
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

	if terminals[pc.sessionID] != nil {
		delete(terminals, pc.sessionID)
	}

	if pc.conn != nil {
		err := pc.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(5*time.Second),
		)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to write close message to pty websocket.")
			return
		}

		err = pc.conn.Close()
		if err != nil {
			log.Debug().Err(err).Msg("Failed to close pty websocket connection.")
		}
	}

	log.Debug().Msg("Websocket connection for pty has been closed.")
}

// recovery reconnects the WebSocket while keeping the PTY session alive.
// Note: recovery doesn't close the existing conn explicitly to avoid breaking the session.
// The goal is to replace a broken connection, not perform a graceful shutdown.
func (pc *PtyClient) recovery() error {
	ctx, cancel := context.WithTimeout(context.Background(), maxRecoveryTimeout)
	defer cancel()

	retryBackoff := backoff.NewExponentialBackOff()
	retryBackoff.InitialInterval = 1 * time.Second
	retryBackoff.MaxInterval = 30 * time.Second
	retryBackoff.MaxElapsedTime = 0 // until ctx timeout
	retryBackoff.RandomizationFactor = 0

	operation := func() error {
		select {
		case <-ctx.Done():
			log.Error().Msg("PTY recovery aborted: timeout reached.")
			return backoff.Permanent(ctx.Err())
		default:
			data := map[string]interface{}{
				"session": pc.sessionID,
			}
			body, statusCode, err := pc.apiSession.Post(reissuePtyWebsocketURL, data, 5)
			if err != nil || statusCode != http.StatusCreated {
				nextInterval := retryBackoff.NextBackOff()
				log.Warn().Err(err).Msgf("Failed to reissue pty websocket (status: %d), will try again in %ds.", statusCode, int(nextInterval.Seconds()))
				return fmt.Errorf("reissue failed: %w", err)
			}

			var resp struct {
				WebsocketURL string `json:"websocket_url"`
			}
			if err = json.Unmarshal(body, &resp); err != nil {
				log.Warn().Err(err).Msg("Failed to parse reissue response.")
				return fmt.Errorf("unmarshal error: %w", err)
			}
			pc.url = strings.Replace(config.GlobalSettings.ServerURL, "http", "ws", 1) + resp.WebsocketURL

			dialer := websocket.Dialer{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: !config.GlobalSettings.SSLVerify,
				},
			}
			conn, _, err := dialer.Dial(pc.url, pc.requestHeader)
			if err != nil {
				log.Warn().Err(err).Msg("PTY websocket reconnection failed.")
				return err
			}

			pc.conn = conn
			log.Info().Msg("PTY WebSocket reconnected successfully.")
			return nil
		}
	}

	err := backoff.Retry(operation, backoff.WithContext(retryBackoff, ctx))
	if err != nil {
		return err
	}

	return nil
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
