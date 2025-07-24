package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// Level define o nível de log
type Level int

const (
	// DEBUG é usado para informações detalhadas de depuração
	DEBUG Level = iota
	// INFO é usado para mensagens informativas gerais
	INFO
	// WARN é usado para situações não críticas mas que merecem atenção
	WARN
	// ERROR é usado para erros que afetam o funcionamento normal
	ERROR
	// FATAL é usado para erros severos que interrompem a aplicação
	FATAL
)

// String retorna a representação em string do nível de log
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger implementa um logger estruturado
type Logger struct {
	level      Level
	enableJSON bool
}

// New cria uma nova instância do Logger
func New(level Level, enableJSON bool) *Logger {
	return &Logger{
		level:      level,
		enableJSON: enableJSON,
	}
}

// SetLevel define o nível de log
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// log gera uma entrada de log no nível especificado
func (l *Logger) log(level Level, message string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	if fields == nil {
		fields = make(map[string]interface{})
	}

	// Adiciona timestamp e nível ao log
	fields["timestamp"] = time.Now().Format(time.RFC3339)
	fields["level"] = level.String()
	fields["message"] = message

	var output string
	if l.enableJSON {
		jsonData, err := json.Marshal(fields)
		if err != nil {
			// Fallback para log simples em caso de erro
			output = fmt.Sprintf("[%s] %s: %s", level.String(), time.Now().Format(time.RFC3339), message)
		} else {
			output = string(jsonData)
		}
	} else {
		// Formato simples para logs não-JSON
		output = fmt.Sprintf("[%s] %s: %s", level.String(), time.Now().Format(time.RFC3339), message)

		// Adiciona os campos extras
		for k, v := range fields {
			if k != "timestamp" && k != "level" && k != "message" {
				output += fmt.Sprintf(" %s=%v", k, v)
			}
		}
	}

	log.Println(output)

	// Em caso de erro fatal, finaliza a aplicação
	if level == FATAL {
		os.Exit(1)
	}
}

// Debug gera um log de nível DEBUG
func (l *Logger) Debug(message string, fields map[string]interface{}) {
	l.log(DEBUG, message, fields)
}

// Info gera um log de nível INFO
func (l *Logger) Info(message string, fields map[string]interface{}) {
	l.log(INFO, message, fields)
}

// Warn gera um log de nível WARN
func (l *Logger) Warn(message string, fields map[string]interface{}) {
	l.log(WARN, message, fields)
}

// Error gera um log de nível ERROR
func (l *Logger) Error(message string, fields map[string]interface{}) {
	l.log(ERROR, message, fields)
}

// Fatal gera um log de nível FATAL e finaliza a aplicação
func (l *Logger) Fatal(message string, fields map[string]interface{}) {
	l.log(FATAL, message, fields)
}
