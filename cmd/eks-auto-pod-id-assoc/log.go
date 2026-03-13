package main

import (
	"fmt"
	"log/slog"
	"os"
)

func setupLogging(levelStr string, isJSON bool) {
	// If the user wants JSON, we MUST override the default handler,
	// regardless of the level.
	if isJSON {
		level := getLevel(levelStr)
		handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
		slog.SetDefault(slog.New(handler))
		return
	}

	// If not JSON, but they want a non-info level, we override with Text.
	if levelStr != "" && levelStr != "info" {
		level := getLevel(levelStr)
		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
		slog.SetDefault(slog.New(handler))
		return
	}

	// Otherwise: stay with the default behavior.
}

func getLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func fatalf(format string, args ...any) {
	slog.Error("FATAL: " + fmt.Sprintf(format, args...))
	os.Exit(1)
}

func infof(format string, args ...any) {
	slog.Info(fmt.Sprintf(format, args...))
}

func errorf(format string, args ...any) {
	slog.Error(fmt.Sprintf(format, args...))
}

func debugf(format string, args ...any) {
	slog.Debug(fmt.Sprintf(format, args...))
}
