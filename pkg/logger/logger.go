package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alpacanetworks/alpamon/pkg/scheduler"
	"github.com/alpacanetworks/alpamon/pkg/version"
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

	recordWriter := &logRecordWriter{}

	var output io.Writer
	// In development, log to console; in production, log to file
	if version.Version == "dev" {
		output = zerolog.MultiLevelWriter(newPrettyWriter(os.Stderr), recordWriter)
	} else {
		output = zerolog.MultiLevelWriter(newPrettyWriter(logFile), recordWriter)
	}

	log.Logger = zerolog.New(output).With().Timestamp().Caller().Logger()
	return logFile
}

func newPrettyWriter(out io.Writer) zerolog.ConsoleWriter {
	return zerolog.ConsoleWriter{
		Out:          out,
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

// logRecordFileHandlers defines log level thresholds for specific files.
// Only files listed here will have their logs sent to the remote server.
// Logs from files not listed will be ignored entirely.
// Logs below the specified level for a listed file will also be ignored.
var logRecordFileHandlers = map[string]int{
	"command.go": 30,
	"commit.go":  20,
	"pty.go":     30,
	"shell.go":   30,
}

func (w *logRecordWriter) Write(p []byte) (n int, err error) {
	var entry zerologEntry
	err = json.Unmarshal(p, &entry)
	if err != nil {
		return n, err
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

	levelThreshold, exists := logRecordFileHandlers[callerFileName]
	if !exists {
		return len(p), nil
	}

	if convertLevelToNumber(entry.Level) < levelThreshold {
		return len(p), nil
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
