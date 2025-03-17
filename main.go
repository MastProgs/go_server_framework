package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go_server_framework/config"
	"go_server_framework/router"
)

func main() {
	cfg := config.GetConfig()

	r := router.SetupRouter()
	jsonLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 워커 풀 시작
	router.Pool.Start()

	port := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:    port,
		Handler: r,
	}

	// 서버를 고루틴에서 시작
	go func() {
		jsonLogger.Info("Server is running on %s port...", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			jsonLogger.Info("HTTP server ListenAndServer", "error", err)
			// 여기서 서버 재시작 로직을 구현할 수 있습니다.
		}
	}()

	// 종료 시그널을 기다림
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	jsonLogger.Info("Shutting down server...")

	// 서버 종료를 위한 컨텍스트 생성
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 서버 종료
	if err := srv.Shutdown(ctx); err != nil {
		jsonLogger.Info("Server Shutdown", "error", err)
	}

	// 워커 풀 종료
	router.Pool.Stop()

	jsonLogger.Info("Server exiting")
}
