package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"vibe-studio/backend/api/middleware"
	"vibe-studio/backend/api/router"
	"vibe-studio/backend/conf"
	"vibe-studio/backend/infra"
	"vibe-studio/backend/pkg/logger"
)

func main() {
	// 加载 .env（不存在则忽略，使用 conf 里的缺省值）。
	_ = godotenv.Load()
	logger.Init() // 配置 slog 默认 logger

	cfg := conf.Load()
	deps := infra.Init(cfg)

	// 标准库 ServeMux（Go 1.22+ 支持 "METHOD /path/{id}" 路由）。
	mux := http.NewServeMux()
	router.Register(mux, deps)

	// 全局中间件链（经典 func(http.Handler) http.Handler）：先列的在最外层。
	handler := middleware.Chain(mux,
		middleware.Recovery(),
		middleware.RequestID(),
		middleware.CORS(),
		middleware.AccessLog(),
	)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen error", "err", err)
			os.Exit(1)
		}
	}()

	// 优雅退出。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
