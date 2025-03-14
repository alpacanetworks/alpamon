package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alpacanetworks/alpamon-go/pkg/config"
	"github.com/alpacanetworks/alpamon-go/pkg/scheduler"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/cenkalti/backoff"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"time"
)

const (
	minConnectInterval    = 5 * time.Second
	maxConnectInterval    = 60 * time.Second
	ConnectionReadTimeout = 35 * time.Minute
	maxRetryTimeout       = 3 * 24 * time.Hour

	eventCommandAckURL = "/api/events/commands/%s/ack/"
	eventCommandFinURL = "/api/events/commands/%s/fin/"
)

type WebsocketClient struct {
	Conn             *websocket.Conn
	requestHeader    http.Header
	apiSession       *scheduler.Session
	RestartRequested bool
	QuitChan         chan struct{}
}

func NewWebsocketClient(session *scheduler.Session) *WebsocketClient {
	headers := http.Header{
		"Authorization": {fmt.Sprintf(`id="%s", key="%s"`, config.GlobalSettings.ID, config.GlobalSettings.Key)},
		"Origin":        {config.GlobalSettings.ServerURL},
		"User-Agent":    {utils.GetUserAgent("alpamon")},
	}

	return &WebsocketClient{
		requestHeader:    headers,
		apiSession:       session,
		RestartRequested: false,
		QuitChan:         make(chan struct{}),
	}
}

func (wc *WebsocketClient) RunForever() {
	wc.Connect()
	defer wc.Close()

	for {
		select {
		case <-wc.QuitChan:
			return
		default:
			err := wc.Conn.SetReadDeadline(time.Now().Add(ConnectionReadTimeout))
			if err != nil {
				wc.CloseAndReconnect()
			}
			_, message, err := wc.ReadMessage()
			if err != nil {
				wc.CloseAndReconnect()
			}
			// Sends "ping" query for Alpacon to verify WebSocket session status without error handling.
			_ = wc.SendPingQuery()
			wc.commandRequestHandler(message)
		}
	}
}

func (wc *WebsocketClient) SendPingQuery() error {
	pingQuery := map[string]string{"query": "ping"}
	err := wc.writeJSON(pingQuery)
	if err != nil {
		return err
	}

	return nil
}

func (wc *WebsocketClient) ReadMessage() (messageType int, message []byte, err error) {
	messageType, message, err = wc.Conn.ReadMessage()
	if err != nil {
		return 0, nil, err
	}

	return messageType, message, nil
}

func (wc *WebsocketClient) Connect() {
	log.Info().Msgf("Connecting to websocket at %s...", config.GlobalSettings.WSPath)

	ctx, cancel := context.WithTimeout(context.Background(), maxRetryTimeout)
	defer cancel()

	wsBackoff := backoff.NewExponentialBackOff()
	wsBackoff.InitialInterval = minConnectInterval
	wsBackoff.MaxInterval = maxConnectInterval
	wsBackoff.MaxElapsedTime = 0      // No time limit for retries (infinite retry)
	wsBackoff.RandomizationFactor = 0 // Retry forever

	operation := func() error {
		select {
		case <-ctx.Done():
			log.Error().Msg("Maximum retry duration reached. Shutting down.")
			return ctx.Err()
		default:
			conn, _, err := websocket.DefaultDialer.Dial(config.GlobalSettings.WSPath, wc.requestHeader)
			if err != nil {
				nextInterval := wsBackoff.NextBackOff()
				log.Debug().Err(err).Msgf("Failed to connect to %s, will try again in %ds.", config.GlobalSettings.WSPath, int(nextInterval.Seconds()))
				return err
			}

			wc.Conn = conn
			log.Debug().Msg("Backhaul connection established.")
			return nil
		}
	}

	err := backoff.Retry(operation, backoff.WithContext(wsBackoff, ctx))
	if err != nil {
		os.Exit(1)
		return
	}
}

func (wc *WebsocketClient) CloseAndReconnect() {
	wc.Close()
	wc.Connect()
}

// Cleanly close the websocket connection by sending a close message
// Do not close quitChan, as the purpose here is to disconnect the WebSocket,
// not to terminate RunForever.
func (wc *WebsocketClient) Close() {
	if wc.Conn != nil {
		err := wc.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Debug().Err(err).Msg("Failed to write close message to websocket.")
		}
		_ = wc.Conn.Close()
	}
}

func (wc *WebsocketClient) Quit() {
	wc.Close()
	close(wc.QuitChan)
}

func (wc *WebsocketClient) Restart() {
	wc.RestartRequested = true
	wc.Quit()
}

func (wc *WebsocketClient) commandRequestHandler(message []byte) {
	var content Content
	var data CommandData

	if len(message) == 0 {
		return
	}

	err := json.Unmarshal(message, &content)
	if err != nil {
		log.Error().Err(err).Msgf("Inappropriate message: %s", string(message))
		return
	}

	if content.Command.Data != "" {
		err = json.Unmarshal([]byte(content.Command.Data), &data)
		if err != nil {
			log.Error().Err(err).Msgf("Inappropriate message: %s", string(message))
			return
		}
	}

	switch content.Query {
	case "command":
		scheduler.Rqueue.Post(fmt.Sprintf(eventCommandAckURL, content.Command.ID),
			nil,
			10,
			time.Time{},
		)
		commandRunner := NewCommandRunner(wc, content.Command, data)
		go commandRunner.Run()
	case "quit":
		log.Debug().Msgf("Quit requested for reason: %s", content.Reason)
		wc.Quit()
	case "reconnect":
		log.Debug().Msgf("Reconnect requested for reason: %s", content.Reason)
		wc.Close()
	default:
		log.Warn().Msgf("Not implemented query: %s", content.Query)
	}
}

func (wc *WebsocketClient) writeJSON(data interface{}) error {
	err := wc.Conn.WriteJSON(data)
	if err != nil {
		log.Debug().Err(err).Msgf("Failed to write json data to websocket.")
		return err
	}
	return nil
}
