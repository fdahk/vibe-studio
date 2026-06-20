// Package logger 配置标准库 log/slog 作为全局结构化日志（零第三方依赖）。
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Init 按 LOG_LEVEL 环境变量设置 slog 默认 logger（text handler，输出到 stdout）。
func Init() {
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(os.Getenv("LOG_LEVEL"))})
	slog.SetDefault(slog.New(h))
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
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
