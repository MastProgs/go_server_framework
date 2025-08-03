package packet_processor

import (
	"go_server_framework/loghandle"
	"go_server_framework/network"
	"go_server_framework/protocol"
	"sync"
	"time"
)

// 패킷 프로세서 (전체 패킷 처리 시스템 관리)
type PacketProcessor struct {
	completionQueue *CompletionQueue
	gameRouter      *GameRouter
	connectionMgr   *network.ConnectionManager

	// 설정
	queueSize   int
	workerCount int

	// 상태
	isRunning bool
	startTime time.Time

	// 제어
	shutdownChan chan struct{}
	wg           sync.WaitGroup
}

// 프로세서 설정
type ProcessorConfig struct {
	QueueSize   int
	WorkerCount int
}

func NewPacketProcessor(config ProcessorConfig, connectionMgr *network.ConnectionManager) *PacketProcessor {
	gameRouter := NewGameRouter()
	completionQueue := NewCompletionQueue(config.QueueSize, config.WorkerCount, gameRouter)

	return &PacketProcessor{
		completionQueue: completionQueue,
		gameRouter:      gameRouter,
		connectionMgr:   connectionMgr,
		queueSize:       config.QueueSize,
		workerCount:     config.WorkerCount,
		shutdownChan:    make(chan struct{}),
	}
}

// 프로세서 시작
func (pp *PacketProcessor) Start() error {
	loghandle.Info("패킷 프로세서 시작")

	// 게임 라우터 시작
	if err := pp.gameRouter.Start(); err != nil {
		return err
	}

	// 완료 큐 시작
	pp.completionQueue.Start()

	// 패킷 이벤트 처리 고루틴 시작
	pp.wg.Add(1)
	go pp.packetEventLoop()

	pp.isRunning = true
	pp.startTime = time.Now()

	loghandle.Info("패킷 프로세서 시작 완료")
	return nil
}

// 프로세서 종료
func (pp *PacketProcessor) Stop() {
	if !pp.isRunning {
		return
	}

	loghandle.Info("패킷 프로세서 종료 시작")

	pp.isRunning = false
	close(pp.shutdownChan)

	// 완료 큐 종료
	pp.completionQueue.Stop()

	// 게임 라우터 종료
	pp.gameRouter.Stop()

	pp.wg.Wait()

	loghandle.Info("패킷 프로세서 종료 완료")
}

// 패킷 이벤트 처리 루프
func (pp *PacketProcessor) packetEventLoop() {
	defer pp.wg.Done()

	packetChan := pp.connectionMgr.GetPacketChannel()

	for {
		select {
		case event := <-packetChan:
			pp.handlePacketEvent(event)

		case <-pp.shutdownChan:
			return
		}
	}
}

// 패킷 이벤트 처리
func (pp *PacketProcessor) handlePacketEvent(event *network.PacketEvent) {
	// 완료 큐에 패킷 수신 이벤트 전송
	if err := pp.completionQueue.PostPacketReceived(event.Connection, event.Packet); err != nil {
		loghandle.Error("완료 큐에 패킷 이벤트 전송 실패: %v", err)

		// 큐가 가득 찬 경우 연결 끊기
		if err == ErrQueueFull {
			event.Connection.Close()
		}
	}
}

// 게임 서비스 등록
func (pp *PacketProcessor) RegisterGameService(service GameService) error {
	return pp.gameRouter.RegisterGameService(service)
}

// 게임 서비스 제거
func (pp *PacketProcessor) UnregisterGameService(gameID protocol.GameID) error {
	return pp.gameRouter.UnregisterGameService(gameID)
}

// 게임 라우터 조회
func (pp *PacketProcessor) GetGameRouter() *GameRouter {
	return pp.gameRouter
}

// 완료 큐 조회
func (pp *PacketProcessor) GetCompletionQueue() *CompletionQueue {
	return pp.completionQueue
}

// 프로세서 상태
func (pp *PacketProcessor) IsRunning() bool {
	return pp.isRunning
}

// 프로세서 통계
func (pp *PacketProcessor) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"is_running":   pp.isRunning,
		"queue_size":   pp.queueSize,
		"worker_count": pp.workerCount,
		"uptime":       time.Since(pp.startTime).String(),
	}

	// 완료 큐 통계 추가
	queueStats := pp.completionQueue.GetStats()
	for k, v := range queueStats {
		stats["queue_"+k] = v
	}

	// 게임 라우터 통계 추가
	routerStats := pp.gameRouter.GetStats()
	for k, v := range routerStats {
		stats["router_"+k] = v
	}

	return stats
}

// 성능 모니터링 정보
func (pp *PacketProcessor) GetPerformanceInfo() map[string]interface{} {
	return map[string]interface{}{
		"queue_utilization": float64(pp.completionQueue.GetQueueSize()) / float64(pp.queueSize) * 100,
		"is_queue_full":     pp.completionQueue.IsFull(),
		"worker_count":      pp.workerCount,
		"uptime_seconds":    time.Since(pp.startTime).Seconds(),
	}
}
