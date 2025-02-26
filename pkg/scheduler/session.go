package scheduler

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/config"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/rs/zerolog/log"
)

const (
	checkSessionURL = "/api/servers/servers/-/"
	MaxRetryTimeout = 3 * 24 * time.Hour
)

func InitSession() *Session {
	session := &Session{
		BaseURL: config.GlobalSettings.ServerURL,
	}

	client := http.Client{}

	tlsConfig := &tls.Config{}
	if config.GlobalSettings.CaCert != "" {
		caCertPool := x509.NewCertPool()
		caCert, err := os.ReadFile(config.GlobalSettings.CaCert)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to read CA certificate")
		}
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	tlsConfig.InsecureSkipVerify = config.GlobalSettings.SSLVerify
	client.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	session.Client = &client
	session.authorization = fmt.Sprintf(`id="%s", key="%s"`, config.GlobalSettings.ID, config.GlobalSettings.Key)

	return session
}

func (session *Session) CheckSession() bool {
	timeout := config.MinConnectInterval
	ctx, cancel := context.WithTimeout(context.Background(), MaxRetryTimeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			log.Error().Msg("Maximum retry duration reached. Shutting down.")
			os.Exit(1)
		default:
			resp, _, err := session.Get(checkSessionURL, 5)
			if err != nil {
				log.Debug().Err(err).Msgf("Failed to connect to %s, will try again in %ds", config.GlobalSettings.ServerURL, int(timeout.Seconds()))
				time.Sleep(timeout)
				timeout *= 2
				if timeout > config.MaxConnectInterval {
					timeout = config.MaxConnectInterval
				}
				continue
			}

			var response map[string]interface{}
			err = json.Unmarshal(resp, &response)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to unmarshal JSON")
				continue
			}

			if commissioned, ok := response["commissioned"].(bool); ok {
				return commissioned
			} else {
				log.Error().Msg("Unable to find 'commissioned' field in the response")
				continue
			}
		}
	}
}

func (session *Session) newRequest(method, url string, rawBody interface{}) (*http.Request, error) {
	var body io.Reader
	if rawBody != nil {
		switch v := rawBody.(type) {
		case string:
			body = strings.NewReader(v)
		case []byte:
			body = bytes.NewReader(v)
		default:
			jsonBody, err := json.Marshal(rawBody)
			if err != nil {
				return nil, err
			}
			body = bytes.NewReader(jsonBody)
		}
	}

	return http.NewRequest(method, utils.JoinPath(session.BaseURL, url), body)
}

func (session *Session) do(req *http.Request, timeout time.Duration) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(req.Context(), timeout*time.Second)
	defer cancel()

	req = req.WithContext(ctx)

	req.Header.Set("Authorization", session.authorization)
	req.Header.Set("User-Agent", utils.GetUserAgent())

	if req.Method == http.MethodPost || req.Method == http.MethodPut || req.Method == http.MethodPatch {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := session.Client.Do(req)
	if err != nil {
		return nil, 0, err
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return body, resp.StatusCode, nil
}

func (session *Session) Request(method, url string, rawBody interface{}, timeout time.Duration) ([]byte, int, error) {
	req, err := session.newRequest(method, url, rawBody)
	if err != nil {
		return nil, 0, err
	}

	resp, statusCode, err := session.do(req, timeout)
	if err != nil {
		return nil, 0, err
	}

	return resp, statusCode, nil
}

func (session *Session) Get(url string, timeout time.Duration) ([]byte, int, error) {
	req, err := session.newRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}

	return session.do(req, timeout)
}

func (session *Session) Post(url string, rawBody interface{}, timeout time.Duration) ([]byte, int, error) {
	req, err := session.newRequest(http.MethodPost, url, rawBody)
	if err != nil {
		return nil, 0, err
	}

	return session.do(req, timeout)
}

func (session *Session) Put(url string, rawBody interface{}, timeout time.Duration) ([]byte, int, error) {
	req, err := session.newRequest(http.MethodPut, url, rawBody)
	if err != nil {
		return nil, 0, err
	}

	return session.do(req, timeout)
}

func (session *Session) Patch(url string, rawBody interface{}, timeout time.Duration) ([]byte, int, error) {
	req, err := session.newRequest(http.MethodPatch, url, rawBody)
	if err != nil {
		return nil, 0, err
	}

	return session.do(req, timeout)
}

func (session *Session) MultipartRequest(url string, body bytes.Buffer, contentType string, timeout time.Duration) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		return nil, 0, err
	}

	ctx, cancel := context.WithTimeout(req.Context(), timeout*time.Second)
	defer cancel()

	req = req.WithContext(ctx)

	req.Header.Set("Authorization", session.authorization)
	req.Header.Set("User-Agent", utils.GetUserAgent())
	req.Header.Set("Content-Type", contentType)

	resp, err := session.Client.Do(req)
	if err != nil {
		return nil, 0, err
	}

	defer func() { _ = resp.Body.Close() }()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return responseBody, resp.StatusCode, nil
}
