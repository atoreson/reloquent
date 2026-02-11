package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/reloquent/reloquent/internal/config"
)

// Setup initializes the logger with file and stdout output.
func Setup(level, directory string) (*slog.Logger, error) {
	if directory == "" {
		directory = config.ExpandHome("~/.reloquent/logs/")
	} else {
		directory = config.ExpandHome(directory)
	}

	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, fmt.Errorf("creating log directory: %w", err)
	}

	filename := fmt.Sprintf("reloquent-%s.log", time.Now().Format("2006-01-02"))
	logPath := filepath.Join(directory, filename)

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}

	writer := io.MultiWriter(os.Stdout, file)

	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: logLevel,
	})

	return slog.New(handler), nil
}
