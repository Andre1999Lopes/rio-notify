package logger

import (
	"log/slog"
	"os"
	"strings"
)

type Logger struct {
	*slog.Logger
}

func New(level string) *Logger {
	logLevel := parseLevel(level)
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "cpf" {
				return slog.Attr{}
			}

			if a.Key == "webhook_secret" {
				return slog.Attr{}
			}

			if a.Key == "token" || a.Key == "jwt" {
				return slog.Attr{}
			}

			if a.Key == "user_hash" && a.Value.Kind() == slog.KindString {
				hash := a.Value.String()
				if len(hash) > 8 {
					a.Value = slog.StringValue(hash[:8] + "...")
				}
			}

			return a
		},
	})

	return &Logger{
		Logger: slog.New(handler),
	}
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (l *Logger) WithError(err error) *slog.Logger {
	return l.With("error", err.Error())
}

func (l *Logger) WithRequestId(requestId string) *slog.Logger {
	return l.With("request_id", requestId)
}

func (l *Logger) WithUser(userHash string) *slog.Logger {
	return l.With("user_hash", userHash)
}

func (l *Logger) WithCallId(callId string) *slog.Logger {
	return l.With("call_id", callId)
}

func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}
