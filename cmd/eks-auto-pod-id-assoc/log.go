package main

import (
	"fmt"
	"log/slog"
	"os"
)

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
