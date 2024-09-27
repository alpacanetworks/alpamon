package logger

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"strings"
	"time"
)

const (
	logDir = "/var/log/alpamon"
)

func InitLogger() *os.File {
	fileName := fmt.Sprintf("%s/alpamon.log", logDir)
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		fileName = "alpamon.log"
	}

	logFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open log file")
	}

	consoleOutput := zerolog.ConsoleWriter{
		Out:          os.Stderr,
		TimeFormat:   time.RFC3339,
		TimeLocation: time.Local,
		FormatLevel: func(i interface{}) string {
			return "[" + strings.ToUpper(i.(string)) + "]"
		},
		FormatMessage: func(i interface{}) string {
			return " " + i.(string)
		},
		FormatFieldName: func(i interface{}) string {
			return "(" + i.(string) + ")"
		},
		FormatFieldValue: func(i interface{}) string {
			return i.(string)
		},
	}

	multi := zerolog.MultiLevelWriter(consoleOutput, logFile)
	log.Logger = zerolog.New(multi).With().Timestamp().Caller().Logger()

	return logFile
}
