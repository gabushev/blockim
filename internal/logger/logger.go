package logger

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
)

type Config struct {
	Level       string
	Output      string
	Pretty      bool
	RemoteURL   string
	RemoteToken string
}

type remoteWriter struct {
	url    string
	token  string
	client *http.Client
}

func (w *remoteWriter) Write(p []byte) (n int, err error) {
	req, err := http.NewRequest("POST", w.url, bytes.NewReader(p))
	if err != nil {
		return 0, err
	}

	if w.token != "" {
		req.Header.Set("Authorization", "Bearer "+w.token)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, err
	}

	return len(p), nil
}

var logger *slog.Logger

func Setup(cfg Config) error {
	// ыet default level if not specified
	level := slog.LevelInfo
	if cfg.Level != "" {
		switch cfg.Level {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		default:
			return nil
		}
	}

	var output io.Writer = os.Stdout // default to stdout

	switch cfg.Output {
	case "stdout", "":
	case "remote":
		if cfg.RemoteURL != "" {
			rw := &remoteWriter{
				url:    cfg.RemoteURL,
				token:  cfg.RemoteToken,
				client: &http.Client{Timeout: 5 * time.Second},
			}
			output = io.MultiWriter(os.Stdout, rw)
		}
	default:
		file, err := os.OpenFile(cfg.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		output = file
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	var handler slog.Handler
	if cfg.Pretty {
		handler = slog.NewTextHandler(output, opts)
	} else {
		handler = slog.NewJSONHandler(output, opts)
	}

	logger = slog.New(handler)
	slog.SetDefault(logger)

	return nil
}

func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}

func Fatal(msg string, args ...any) {
	logger.Error(msg, args...)
	os.Exit(1)
}

func Get() *slog.Logger {
	return logger
}

func WithContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return logger
	}
	if gc, ok := ctx.(*gin.Context); ok {
		if reqID := gc.GetHeader("X-Request-ID"); reqID != "" {
			return logger.With(slog.String("request_id", reqID))
		}
	}

	return logger
}

func GinMiddleware() gin.HandlerFunc {
	return sloggin.New(logger)
}
