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
	MinConnectInterval = 5 * time.Second
	MaxConnectInterval = 60 * time.Second

	eventCommandAckURL = "/api/events/commands/%s/ack/"
	eventCommandFinURL = "/api/events/commands/%s/fin/"
)

type WebsocketClient struct {
	conn          *websocket.Conn
	requestHeader http.Header
	apiSession    *scheduler.Session
	quitChan      chan struct{}
}

func NewWebsocketClient(session *scheduler.Session) *WebsocketClient {
	headers := http.Header{
		"Authorization": {fmt.Sprintf(`id="%s", key="%s"`, config.GlobalSettings.ID, config.GlobalSettings.Key)},
		"Origin":        {config.GlobalSettings.ServerURL},
	}

	return &WebsocketClient{
		requestHeader: headers,
		apiSession:    session,
		quitChan:      make(chan struct{}),
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
			wc.commandRequestHandler(message)
		}
	}
}

func (wc *WebsocketClient) readMessage() (messageType int, message []byte, err error) {
	messageType, message, err = wc.conn.ReadMessage()
	if err != nil {
		return 0, nil, err
	}

	return messageType, message, nil
}

func (wc *WebsocketClient) connect() {
	wsBackoff := backoff.NewExponentialBackOff()
	wsBackoff.InitialInterval = MinConnectInterval
	wsBackoff.MaxInterval = MaxConnectInterval
	wsBackoff.MaxElapsedTime = 0      // No time limit for retries (infinite retry)
	wsBackoff.RandomizationFactor = 0 // Retry forever

	err := backoff.Retry(func() error {
		conn, _, err := websocket.DefaultDialer.Dial(config.GlobalSettings.WSPath, wc.requestHeader)
		if err != nil {
			nextInterval := wsBackoff.NextBackOff()
			log.Debug().Err(err).Msgf("Failed to connect to %s, will try again in %ds", config.GlobalSettings.WSPath, int(nextInterval.Seconds()))
			return err
		}

		wc.conn = conn
		log.Debug().Msg("Backhaul connection established")
		return nil
	}, wsBackoff)

	if err != nil {
		log.Error().Err(err).Msgf("Could not connect to %s: terminated unexpectedly", config.GlobalSettings.WSPath)
		return
	}

	return
}

func (wc *WebsocketClient) closeAndReconnect() {
	wc.close()
	wc.connect()
}

// TODO
func (wc *WebsocketClient) writeJSON(v interface{}) {}

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

func (wc *WebsocketClient) commandRequestHandler(message []byte) {
	var content Content
	var data CommandData
	err := json.Unmarshal(message, &content)
	if err != nil {
		log.Error().Err(err).Msgf("Anappropriate message: %s", message)
		return
	}

	if content.Command.Data != "" {
		err = json.Unmarshal([]byte(content.Command.Data), &data)
		if err != nil {
			log.Error().Err(err).Msgf("Anappropriate message: %s", message)
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
