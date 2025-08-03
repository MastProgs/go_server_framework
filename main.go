package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go_server_framework/config"
	"go_server_framework/database"
	"go_server_framework/game_services"
	appInit "go_server_framework/init"
	"go_server_framework/loghandle"
	"go_server_framework/network"
	"go_server_framework/packet_processor"
	"go_server_framework/protocol"
)

func main() {
	// 초기화
	if err := appInit.InitAll(); err != nil {
		loghandle.Error("초기화 실패: %v", err)
		return
	}

	// 프로토콜 정보 출력
	protocol.PrintAllPackets()

	// 설정 조회
	cfg := config.GetConfig()

	err := appInit.PreInit()
	if err != nil {
		loghandle.Error("애플리케이션 PreInit 실패: %v", err)
		os.Exit(1)
	}

	// TCP 게임서버 설정
	serverConfig := network.ServerConfig{
		Address:        fmt.Sprintf(":%d", cfg.Server.Port),
		MaxConnections: 1000, // 기본값
	}

	// 게임서버 생성
	gameServer := network.NewGameServer(serverConfig)

	// 패킷 프로세서 생성
	processorConfig := packet_processor.ProcessorConfig{
		QueueSize:   1000,
		WorkerCount: 10,
	}
	processor := packet_processor.NewPacketProcessor(processorConfig, gameServer.GetConnectionManager())

	// 게임 서비스 매니저 생성 및 서비스 등록
	repo, err := database.NewRepository()
	if err != nil {
		loghandle.Error("데이터베이스 리포지토리 생성 실패: %v", err)
		return
	}
	gameServiceManager := game_services.NewGameServiceManager(processor, repo)
	if err := gameServiceManager.RegisterAllGameServices(); err != nil {
		loghandle.Error("게임 서비스 등록 실패: %v", err)
		return
	}

	// 서버 시작
	loghandle.Info("TCP 게임서버 시작 중...")

	if err := gameServer.Start(); err != nil {
		loghandle.Error("게임서버 시작 실패: %v", err)
		return
	}

	if err := processor.Start(); err != nil {
		loghandle.Error("패킷 프로세서 시작 실패: %v", err)
		gameServer.Stop()
		return
	}

	loghandle.Info("🎮 TCP 게임서버 시작 완료!")
	loghandle.Info("📡 주소: %s", serverConfig.Address)
	loghandle.Info("🔗 최대 연결: %d", serverConfig.MaxConnections)
	loghandle.Info("⚡ 워커 수: %d", processorConfig.WorkerCount)
	loghandle.Info("📦 큐 크기: %d", processorConfig.QueueSize)

	// Graceful shutdown을 위한 채널
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	loghandle.Info("서버가 시작되었습니다. 종료하려면 Ctrl+C를 누르세요.")

	// 종료 신호 대기
	<-quit
	loghandle.Info("서버 종료 신호 수신...")

	// Graceful shutdown
	gameServer.GracefulShutdown(30 * time.Second)
	processor.Stop()
	gameServiceManager.Shutdown()

	loghandle.Info("🎯 TCP 게임서버가 정상적으로 종료되었습니다.")
}
