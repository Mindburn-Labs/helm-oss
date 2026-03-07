package main

import (
	"log/slog"
	"os"
	"strconv"
)

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvRequired(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required environment variable missing", "key", key)
		os.Exit(1)
	}
	return v
}

func getenvIntDefault(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
