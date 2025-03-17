package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	appInit "go_server_framework/init"
	"go_server_framework/loghandle"
)

func main() {
	// 애플리케이션 초기화
	appInit.InitAll()

	port := fmt.Sprintf(":%d", appInit.Config.Server.Port)
	srv := &http.Server{
		Addr:    port,
		Handler: appInit.Router,
	}

	// 서버를 고루틴에서 시작
	go func() {
		// 싱글턴 로거 사용
		loghandle.Info("Server is running on %s port...", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			loghandle.Error("HTTP server ListenAndServe error: %v", err)
			// 여기서 서버 재시작 로직을 구현할 수 있습니다.
		}
	}()

	// 종료 시그널을 기다림
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	loghandle.Info("Shutting down server...")

	// 서버 종료를 위한 컨텍스트 생성
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 서버 종료
	if err := srv.Shutdown(ctx); err != nil {
		loghandle.Error("Server Shutdown error: %v", err)
	}

	// 애플리케이션 종료
	appInit.ShutdownAll()

	loghandle.Info("Server exiting")
}
