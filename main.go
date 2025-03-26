package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go_server_framework/config"
	appInit "go_server_framework/init"
	"go_server_framework/loghandle"
)

func main() {
	// 애플리케이션 초기화
	err := appInit.InitAll()
	if err != nil {
		loghandle.Error("애플리케이션 InitAll 실패: %v", err)
		os.Exit(1)
	}

	Config := config.GetConfig()

	port := fmt.Sprintf(":%d", Config.Server.Port)
	srv := &http.Server{
		Addr:    port,
		Handler: appInit.Router,
	}

	err = appInit.PreInit()
	if err != nil {
		loghandle.Error("애플리케이션 PreInit 실패: %v", err)
		os.Exit(1)
	}

	// 서버 종료를 위한 에러 채널 생성
	errChan := make(chan error, 1)

	// 서버를 고루틴에서 시작
	go func() {
		loghandle.Info("Server is running on %s port...", port)
		if Config.Server.Debug {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				loghandle.Error("HTTP server ListenAndServe error: %v", err)
				errChan <- fmt.Errorf("HTTP 서버 에러: %v", err)
				return
			}
		} else {
			if Config.Server.Certfile != "" && Config.Server.Keyfile != "" {
				// 인증서 파일 존재 여부 확인
				if _, err := os.Stat(Config.Server.Certfile); os.IsNotExist(err) {
					loghandle.Error("인증서 파일을 찾을 수 없습니다: %s", Config.Server.Certfile)
					errChan <- fmt.Errorf("인증서 파일을 찾을 수 없습니다: %s", Config.Server.Certfile)
					return
				}

				// 키 파일 존재 여부 확인
				if _, err := os.Stat(Config.Server.Keyfile); os.IsNotExist(err) {
					loghandle.Error("키 파일을 찾을 수 없습니다: %s", Config.Server.Keyfile)
					errChan <- fmt.Errorf("키 파일을 찾을 수 없습니다: %s", Config.Server.Keyfile)
					return
				}

				if err := srv.ListenAndServeTLS(Config.Server.Certfile, Config.Server.Keyfile); err != nil && err != http.ErrServerClosed {
					loghandle.Error("HTTPS server ListenAndServeTLS error: %v", err)
					errChan <- fmt.Errorf("HTTPS 서버 에러: %v", err)
					return
				}
			} else {
				loghandle.Error("HTTPS server Certfile or Keyfile is not set")
				errChan <- fmt.Errorf("HTTPS 서버 인증서 또는 키 파일이 설정되지 않았습니다")
				return
			}
		}
	}()

	// 종료 시그널을 기다림
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 종료 시그널과 에러 채널을 동시에 처리
	select {
	case <-quit:
		loghandle.Info("정상 종료 시그널을 받았습니다...")
	case err := <-errChan:
		loghandle.Error("서버 에러 발생으로 종료합니다: %v", err)
	}

	// 서버 종료 처리
	loghandle.Info("서버를 종료하는 중...")

	// 서버 종료를 위한 컨텍스트 생성
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 서버 종료
	if err := srv.Shutdown(ctx); err != nil {
		loghandle.Error("Server Shutdown error: %v", err)
	}

	// 애플리케이션 종료
	err = appInit.ShutdownAll()
	if err != nil {
		loghandle.Error("애플리케이션 종료 실패: %v", err)
		os.Exit(1)
	}

	loghandle.Info("서버가 종료되었습니다")
	os.Exit(1) // 에러로 인한 종료는 1을 반환
}
