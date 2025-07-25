package shared

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

type Logger struct {
	level  LogLevel
	json   bool
	logger *log.Logger
}

func NewLogger(level LogLevel, json bool) *Logger {
	return &Logger{
		level:  level,
		json:   json,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

func (l *Logger) Debug(message string, fields map[string]interface{}) {
	if l.level <= DEBUG {
		l.log(DEBUG, message, fields)
	}
}

func (l *Logger) Info(message string, fields map[string]interface{}) {
	if l.level <= INFO {
		l.log(INFO, message, fields)
	}
}

func (l *Logger) Warn(message string, fields map[string]interface{}) {
	if l.level <= WARN {
		l.log(WARN, message, fields)
	}
}

func (l *Logger) Error(message string, fields map[string]interface{}) {
	if l.level <= ERROR {
		l.log(ERROR, message, fields)
	}
}

func (l *Logger) Fatal(message string, fields map[string]interface{}) {
	l.log(ERROR, message, fields)
	os.Exit(1)
}

func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	if l.json {
		l.logJSON(level, message, fields)
	} else {
		l.logText(level, message, fields)
	}
}

func (l *Logger) logJSON(level LogLevel, message string, fields map[string]interface{}) {
	logEntry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     l.levelString(level),
		"message":   message,
	}

	for key, value := range fields {
		logEntry[key] = value
	}

	jsonData, err := json.Marshal(logEntry)
	if err != nil {
		l.logger.Printf("Error marshaling log entry: %v", err)
		return
	}

	l.logger.Println(string(jsonData))
}

func (l *Logger) logText(level LogLevel, message string, fields map[string]interface{}) {
	levelStr := l.levelString(level)
	timestamp := time.Now().Format("2006-01-02T15:04:05Z07:00")

	logMsg := fmt.Sprintf("[%s] %s: %s", timestamp, levelStr, message)

	if len(fields) > 0 {
		fieldsStr := ""
		for key, value := range fields {
			if fieldsStr != "" {
				fieldsStr += ", "
			}
			fieldsStr += fmt.Sprintf("%s=%v", key, value)
		}
		logMsg += fmt.Sprintf(" | %s", fieldsStr)
	}

	l.logger.Println(logMsg)
}

func (l *Logger) levelString(level LogLevel) string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}
