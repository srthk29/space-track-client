package spacetrackclient

import (
	"log/slog"
	"os"
	"strings"
)

type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

func (l LogLevel) Slog() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (l LogLevel) IsValid() bool {
	switch l {
	case LevelDebug, LevelError, LevelWarn, LevelInfo:
		return true
	default:
		return false
	}
}

func getLogLevelEnv() LogLevel {
	loglevel := os.Getenv("LOG_LEVEL")
	loglevel = strings.ToUpper(strings.TrimSpace(loglevel))

	if loglevel == "" {
		return LevelInfo
	}

	if !LogLevel(loglevel).IsValid() {
		return LevelInfo
	}

	return LogLevel(loglevel)
}
