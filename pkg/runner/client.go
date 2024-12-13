package runner

import (
	"encoding/json"
	"fmt"
	"github.com/alpacanetworks/alpamon-go/pkg/config"
	"github.com/alpacanetworks/alpamon-go/pkg/scheduler"
	"github.com/cenkalti/backoff"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"
)

const (
	minConnectInterval = 5 * time.Second
	maxConnectInterval = 60 * time.Second

	eventCommandAckURL = "/api/events/commands/%s/ack/"
	eventCommandFinURL = "/api/events/commands/%s/fin/"
)

type WebsocketClient struct {
	conn             *websocket.Conn
	requestHeader    http.Header
	apiSession       *scheduler.Session
	RestartRequested bool
	quitChan         chan struct{}
}

func NewWebsocketClient(session *scheduler.Session) *WebsocketClient {
	headers := http.Header{
		"Authorization": {fmt.Sprintf(`id="%s", key="%s"`, config.GlobalSettings.ID, config.GlobalSettings.Key)},
		"Origin":        {config.GlobalSettings.ServerURL},
	}

	return &WebsocketClient{
		requestHeader:    headers,
		apiSession:       session,
		RestartRequested: false,
		quitChan:         make(chan struct{}),
	}
}

func (wc *WebsocketClient) RunForever() {
	wc.connect()
	defer wc.close()

	for {
		select {
		case <-wc.quitChan:
			return
		default:
			_, message, err := wc.readMessage()
			if err != nil {
				wc.closeAndReconnect()
			}
			// Sends "ping" query for Alpacon to verify WebSocket session status without error handling.
			_ = wc.sendPingQuery()
			wc.commandRequestHandler(message)
		}
	}
}

func (wc *WebsocketClient) sendPingQuery() error {
	pingQuery := map[string]string{"query": "ping"}
	err := wc.writeJSON(pingQuery)
	if err != nil {
		return err
	}

	return nil
}

func (wc *WebsocketClient) readMessage() (messageType int, message []byte, err error) {
	messageType, message, err = wc.conn.ReadMessage()
	if err != nil {
		return 0, nil, err
	}

	return messageType, message, nil
}

func (wc *WebsocketClient) connect() {
	log.Info().Msgf("Connecting to websocket at %s", config.GlobalSettings.WSPath)

	wsBackoff := backoff.NewExponentialBackOff()
	wsBackoff.InitialInterval = minConnectInterval
	wsBackoff.MaxInterval = maxConnectInterval
	wsBackoff.MaxElapsedTime = 0      // No time limit for retries (infinite retry)
	wsBackoff.RandomizationFactor = 0 // Retry forever

	operation := func() error {
		conn, _, err := websocket.DefaultDialer.Dial(config.GlobalSettings.WSPath, wc.requestHeader)
		if err != nil {
			nextInterval := wsBackoff.NextBackOff()
			log.Debug().Err(err).Msgf("Failed to connect to %s, will try again in %ds", config.GlobalSettings.WSPath, int(nextInterval.Seconds()))
			return err
		}

		wc.conn = conn
		log.Debug().Msg("Backhaul connection established")
		return nil
	}

	err := backoff.Retry(operation, wsBackoff)
	if err != nil {
		log.Error().Err(err).Msg("Unexpected error occurred during backoff")
		return
	}
}

func (wc *WebsocketClient) closeAndReconnect() {
	wc.close()
	wc.connect()
}

// Cleanly close the websocket connection by sending a close message
// Do not close quitChan, as the purpose here is to disconnect the WebSocket,
// not to terminate RunForever.
func (wc *WebsocketClient) close() {
	if wc.conn != nil {
		err := wc.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Debug().Err(err).Msg("Failed to write close message to websocket")
		}
		_ = wc.conn.Close()
	}
}

func (wc *WebsocketClient) quit() {
	wc.close()
	close(wc.quitChan)
}

func (wc *WebsocketClient) restart() {
	wc.RestartRequested = true
	wc.quit()
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
		runner := NewCommandRunner(wc, content.Command, data)
		go runner.Run()
	case "quit":
		log.Debug().Msgf("Quit requested for reason: %s", content.Reason)
		wc.quit()
	case "reconnect":
		log.Debug().Msgf("Reconnect requested for reason: %s", content.Reason)
		wc.close()
	default:
		log.Warn().Msgf("Not implemented query: %s", content.Query)
	}
}

func (wc *WebsocketClient) writeJSON(data interface{}) error {
	err := wc.conn.WriteJSON(data)
	if err != nil {
		log.Debug().Err(err).Msgf("Failed to write json data to websocket")
		return err
	}
	return nil
}
