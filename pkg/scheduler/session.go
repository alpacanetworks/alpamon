package scheduler

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/alpacanetworks/alpamon-go/pkg/config"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	checkSessionURL = "/api/servers/servers/-/"
)

func InitSession() *Session {
	session := &Session{
		BaseURL: config.GlobalSettings.ServerURL,
	}

	client := retryablehttp.NewClient()
	client.RetryMax = 3
	client.RetryWaitMin = 1 * time.Second
	client.RetryWaitMax = 3 * time.Second

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

	client.HTTPClient.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	session.Client = client
	session.authorization = fmt.Sprintf(`id="%s", key="%s"`, config.GlobalSettings.ID, config.GlobalSettings.Key)

	return session
}

func (session *Session) CheckSession() bool {
	timeout := config.MinConnectInterval

	for {
		resp, _, err := session.Get(checkSessionURL, 5)
		if err != nil {
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
			return false
		}
	}
}

func (session *Session) newRequest(method, url string, rawBody interface{}) (*retryablehttp.Request, error) {
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

	return retryablehttp.NewRequest(method, utils.JoinPath(session.BaseURL, url), body)
}

func (session *Session) do(req *retryablehttp.Request, timeout time.Duration) ([]byte, int, error) {
	session.Client.HTTPClient.Timeout = timeout * time.Second
	req.Header.Set("Authorization", session.authorization)

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

func (session *Session) MultipartRequest(url string, body bytes.Buffer, contentType string, timeout time.Duration) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		return nil, 0, err
	}

	session.Client.HTTPClient.Timeout = timeout * time.Second
	req.Header.Set("Authorization", session.authorization)
	req.Header.Set("Content-Type", contentType)

	retryableReq, err := retryablehttp.FromRequest(req)
	if err != nil {
		return nil, 0, err
	}

	resp, err := session.Client.Do(retryableReq)
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
