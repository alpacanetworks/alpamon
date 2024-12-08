package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/scheduler"
	"github.com/alpacanetworks/alpamon-go/pkg/version"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
		_, _ = fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		os.Exit(1)
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

type zerologEntry struct {
	Level   string `json:"level"`
	Time    string `json:"time"`
	Caller  string `json:"caller"`
	Message string `json:"message"`
}

type logRecordWriter struct{}

// remoteLogThresholds defines log level thresholds for specific callers (files).
// Logs below the specified level for a given file will not be sent to the alpacon-server.
// If a file is not listed, all logs will be sent regardless of level.
var remoteLogThresholds = map[string]int{
	"client.go":   30,
	"reporter.go": 40,
	"command.go":  30,
	"commit.go":   30,
	"pty.go":      30,
}

func (w *logRecordWriter) Write(p []byte) (n int, err error) {
	var entry zerologEntry
	err = json.Unmarshal(p, &entry)
	if err != nil {
		return 0, err
	}

	caller := entry.Caller
	if caller == "" {
		return len(p), nil
	}

	lineno := 0
	if parts := strings.Split(caller, ":"); len(parts) > 1 {
		lineno, _ = strconv.Atoi(parts[1])
	}

	callerFileName := getCallerFileName(caller)
	if levelThreshold, ok := remoteLogThresholds[callerFileName]; ok {
		if convertLevelToNumber(entry.Level) < levelThreshold {
			return len(p), nil
		}
	}

	record := logRecord{
		Date:    entry.Time,
		Level:   convertLevelToNumber(entry.Level),
		Program: "alpamon",
		Path:    caller,
		Lineno:  lineno,
		PID:     os.Getpid(),
		Msg:     entry.Message,
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

func getCallerFileName(caller string) string {
	parts := strings.Split(caller, "/")
	if len(parts) > 0 {
		fileWithLine := parts[len(parts)-1]
		fileParts := strings.Split(fileWithLine, ":")
		return fileParts[0]
	}
	return ""
}
