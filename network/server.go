package network

import (
	"go_server_framework/loghandle"
	"sync"
	"time"
)

// TCP 게임 서버
type GameServer struct {
	address         string
	listener        *Listener
	connectionMgr   *ConnectionManager
	
	// 설정
	maxConnections  int
	
	// 상태
	isRunning       bool
	startTime       time.Time
	
	// 제어
	shutdownChan    chan struct{}
	wg              sync.WaitGroup
}

// 서버 설정
type ServerConfig struct {
	Address        string
	MaxConnections int
}

func NewGameServer(config ServerConfig) *GameServer {
	return &GameServer{
		address:        config.Address,
		maxConnections: config.MaxConnections,
		shutdownChan:   make(chan struct{}),
	}
}

// 서버 시작
func (gs *GameServer) Start() error {
	loghandle.Info("게임 서버 시작 중...")
	
	// 연결 관리자 생성 및 시작
	gs.connectionMgr = NewConnectionManager(gs.maxConnections)
	gs.connectionMgr.Start()
	
	// 리스너 생성 및 시작
	listener, err := NewListener(gs.address, gs.connectionMgr)
	if err != nil {
		return err
	}
	gs.listener = listener
	gs.listener.Start()
	
	gs.isRunning = true
	gs.startTime = time.Now()
	
	loghandle.Info("게임 서버 시작 완료: %s (최대 연결: %d)", gs.address, gs.maxConnections)
	
	// 서버 모니터링 고루틴 시작
	gs.wg.Add(1)
	go gs.monitoringLoop()
	
	return nil
}

// 서버 종료
func (gs *GameServer) Stop() {
	if !gs.isRunning {
		return
	}
	
	loghandle.Info("게임 서버 종료 시작...")
	
	gs.isRunning = false
	close(gs.shutdownChan)
	
	// 리스너 종료
	if gs.listener != nil {
		gs.listener.Stop()
	}
	
	// 연결 관리자 종료
	if gs.connectionMgr != nil {
		gs.connectionMgr.Stop()
	}
	
	gs.wg.Wait()
	
	loghandle.Info("게임 서버 종료 완료")
}

// 서버 대기 (블로킹)
func (gs *GameServer) Wait() {
	gs.wg.Wait()
}

// 연결 관리자 조회
func (gs *GameServer) GetConnectionManager() *ConnectionManager {
	return gs.connectionMgr
}

// 서버 상태 조회
func (gs *GameServer) IsRunning() bool {
	return gs.isRunning
}

// 서버 통계
func (gs *GameServer) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"address":    gs.address,
		"is_running": gs.isRunning,
		"uptime":     time.Since(gs.startTime).String(),
		"start_time": gs.startTime,
	}
	
	// 연결 관리자 통계 추가
	if gs.connectionMgr != nil {
		connStats := gs.connectionMgr.GetStats()
		for k, v := range connStats {
			stats[k] = v
		}
	}
	
	return stats
}

// 모니터링 루프
func (gs *GameServer) monitoringLoop() {
	defer gs.wg.Done()
	
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			stats := gs.GetStats()
			loghandle.Info("서버 상태 - 연결: %v, 가동시간: %v", 
				stats["current_connections"], stats["uptime"])
			
		case <-gs.shutdownChan:
			return
		}
	}
}

// 우아한 종료 (Graceful Shutdown)
func (gs *GameServer) GracefulShutdown(timeout time.Duration) {
	loghandle.Info("우아한 종료 시작 (타임아웃: %v)", timeout)
	
	// 새 연결 수락 중지
	if gs.listener != nil {
		gs.listener.Stop()
	}
	
	// 기존 연결들에게 종료 알림
	if gs.connectionMgr != nil {
		// TODO: 클라이언트들에게 서버 종료 알림 패킷 전송
	}
	
	// 타임아웃 내에서 연결들이 자연스럽게 종료되기를 기다림
	done := make(chan struct{})
	go func() {
		gs.Stop()
		close(done)
	}()
	
	select {
	case <-done:
		loghandle.Info("우아한 종료 완료")
	case <-time.After(timeout):
		loghandle.Warn("우아한 종료 타임아웃, 강제 종료")
		gs.Stop()
	}
}

// 특정 연결에 패킷 전송
func (gs *GameServer) SendToConnection(connectionID uint64, packet interface{}) error {
	if gs.connectionMgr == nil {
		return ErrServerNotRunning
	}
	
	_, exists := gs.connectionMgr.GetConnection(connectionID)
	if !exists {
		return ErrConnectionNotFound
	}
	
	// TODO: packet을 protocol.Packet으로 변환 후 전송
	return nil
}

// 브로드캐스트
func (gs *GameServer) Broadcast(packet interface{}) error {
	if gs.connectionMgr == nil {
		return ErrServerNotRunning
	}
	
	// TODO: packet을 protocol.Packet으로 변환 후 브로드캐스트
	return nil
}