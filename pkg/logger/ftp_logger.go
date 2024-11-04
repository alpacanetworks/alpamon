package logger

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type FtpLogger struct {
	log zerolog.Logger
}

// TODO : Send logs to alpamon's Logserver using a Unix domain socket
func NewFtpLogger() FtpLogger {
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

	logger := zerolog.New(consoleOutput).With().Timestamp().Caller().Logger()

	return FtpLogger{
		log: logger,
	}
}

func (l *FtpLogger) Debug() *zerolog.Event {
	return l.log.Debug()
}

func (l *FtpLogger) Info() *zerolog.Event {
	return l.log.Info()
}

func (l *FtpLogger) Warn() *zerolog.Event {
	return l.log.Warn()
}

func (l *FtpLogger) Error() *zerolog.Event {
	return l.log.Error()
}

func (l *FtpLogger) Fatal() *zerolog.Event {
	return l.log.Fatal()
}
