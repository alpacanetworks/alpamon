package config

import (
	"crypto/tls"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/ini.v1"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	configFiles = []string{
		"/etc/alpamon/alpamon.conf",
		filepath.Join(os.Getenv("HOME"), ".alpamon.conf"),
	}

	GlobalSettings Settings
)

const (
	wsPath             = "/ws/servers/backhaul/"
	MinConnectInterval = 5 * time.Second
	MaxConnectInterval = 60 * time.Second
)

func InitSettings(settings Settings) {
	GlobalSettings = settings
}

func LoadConfig() Settings {
	var iniData *ini.File
	var err error
	for _, configFile := range configFiles {
		iniData, err = ini.Load(configFile)
		if err == nil {
			break
		}
	}
	if iniData == nil {
		log.Fatal().Err(err).Msg("Failed to load config file")
	}

	var config Config
	err = iniData.MapTo(&config)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse config file")
	}

	if config.Logging.Debug {
		log.Logger = log.Level(zerolog.DebugLevel)
	}

	valid, settings := validateConfig(config)

	if !valid {
		log.Fatal().Msg("Aborting...")
	}

	return settings
}

func validateConfig(config Config) (bool, Settings) {
	log.Debug().Msg("Validating configuration fields...")

	settings := Settings{
		WSPath:      wsPath,
		UseSSL:      false,
		SSLVerify:   true,
		SSLOpt:      make(map[string]interface{}),
		HTTPThreads: 4,
	}

	valid := true
	val := config.Server.URL
	if strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") {
		if strings.HasSuffix(val, "/") {
			val = val[:len(val)-1]
		}
		settings.ServerURL = val
		settings.WSPath = strings.Replace(val, "http", "ws", 1) + settings.WSPath
		settings.UseSSL = strings.HasPrefix(val, "https://")
	} else {
		log.Error().Msg("Server url is invalid")
		valid = false
	}

	if config.Server.ID != "" && config.Server.Key != "" {
		settings.ID = config.Server.ID
		settings.Key = config.Server.Key
	} else {
		log.Error().Msg("Server ID, KEY is empty")
		valid = false
	}

	if settings.UseSSL {
		settings.SSLVerify = config.SSL.Verify
		caCert := config.SSL.CaCert
		if !settings.SSLVerify {
			log.Warn().Msg(
				"SSL verification is turned off. " +
					"Please be aware that this setting is not appropriate for production use.",
			)
			settings.SSLOpt["cert_reqs"] = &tls.Config{InsecureSkipVerify: true}
		} else if caCert != "" {
			if _, err := os.Stat(caCert); os.IsNotExist(err) {
				log.Error().Msg("Given path for CA certificate does not exist.")
				valid = false
			} else {
				settings.CaCert = caCert
				settings.SSLOpt["ca_certs"] = caCert
			}
		}
	}
	return valid, settings
}
