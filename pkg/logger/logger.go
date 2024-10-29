package logger

import (
	"encoding/json"
	"fmt"
	"github.com/alpacanetworks/alpamon-go/pkg/scheduler"
	"github.com/alpacanetworks/alpamon-go/pkg/version"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	logDir      = "/var/log/alpamon"
	logFileName = "alpamon.log"
	recordURL   = "/api/history/logs/"
)

func InitLogger() *os.File {
	fileName := fmt.Sprintf("%s/%s", logDir, logFileName)
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		fileName = logFileName
	}

	logFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open log file")
	}

	var output io.Writer

	recordWriter := &logRecordWriter{}

	// In development, log to console; in production, log to file
	if version.Version == "dev" {
		consoleWriter := zerolog.ConsoleWriter{
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
		output = zerolog.MultiLevelWriter(consoleWriter, recordWriter)
	} else {
		output = zerolog.MultiLevelWriter(logFile, recordWriter)
	}

	log.Logger = zerolog.New(output).With().Timestamp().Caller().Logger()

	return logFile
}

type logRecord struct {
	Date    string `json:"date"`
	Level   int    `json:"level"`
	Program string `json:"program"`
	Path    string `json:"path"`
	Lineno  int    `json:"lineno"`
	PID     int    `json:"pid"`
	Msg     string `json:"msg"`
}

type logRecordWriter struct{}

func (w *logRecordWriter) Write(p []byte) (n int, err error) {
	var parsedLog map[string]string
	err = json.Unmarshal(p, &parsedLog)
	if err != nil {
		return 0, err
	}

	caller := parsedLog["caller"]
	if caller == "" {
		return len(p), nil
	}

	lineno := 0
	if parts := strings.Split(caller, ":"); len(parts) > 1 {
		lineno, _ = strconv.Atoi(parts[1])
	}

	record := logRecord{
		Date:    time.Now().UTC().Format(time.RFC3339),
		Level:   convertLevelToNumber(parsedLog["level"]),
		Program: "alpamon",
		Path:    caller,
		Lineno:  lineno,
		PID:     os.Getpid(),
		Msg:     parsedLog["message"],
	}

	go func() {
		scheduler.Rqueue.Post(recordURL, record, 90, time.Time{})
	}()

	return len(p), nil
}

// alpacon-server uses Python's logging package, which has different log levels from zerolog.
// This function maps zerolog log levels to Python logging levels.
func convertLevelToNumber(level string) int {
	switch level {
	case "fatal":
		return 50 // CRITICAL, FATAL
	case "error":
		return 40 // ERROR
	case "warn", "warning":
		return 30 // WARNING
	case "info":
		return 20 // INFO
	case "debug":
		return 10 // DEBUG
	default:
		return 0 // NOT SET
	}
}
