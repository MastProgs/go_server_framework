package network

import (
	"fmt"
	"go_server_framework/loghandle"
	"go_server_framework/protocol"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// 연결 관리자
type ConnectionManager struct {
	connections    sync.Map         // map[uint64]*Connection
	idGenerator    uint64           // atomic 연결 ID
	maxConnections int              // 최대 동시 연결 수
	currentCount   int32            // 현재 연결 수
	
	// 이벤트 채널
	connectChan    chan *Connection
	disconnectChan chan uint64
	packetChan     chan *PacketEvent
	
	// 설정
	readBufferSize  int
	writeBufferSize int
	
	// 로깅은 전역 함수 사용
	
	// 제어
	shutdownChan chan struct{}
	wg           sync.WaitGroup
}

// 패킷 이벤트
type PacketEvent struct {
	Connection *Connection
	Packet     protocol.Packet
	Timestamp  time.Time
}

func NewConnectionManager(maxConnections int) *ConnectionManager {
	return &ConnectionManager{
		maxConnections:  maxConnections,
		connectChan:     make(chan *Connection, 100),
		disconnectChan:  make(chan uint64, 100),
		packetChan:      make(chan *PacketEvent, 1000),
		readBufferSize:  4096,
		writeBufferSize: 4096,
		shutdownChan:    make(chan struct{}),
	}
}

// 연결 관리자 시작
func (cm *ConnectionManager) Start() {
	loghandle.Info("연결 관리자 시작")
	
	// 이벤트 처리 고루틴들 시작
	cm.wg.Add(3)
	go cm.connectionEventLoop()
	go cm.disconnectionEventLoop()
	go cm.cleanupLoop()
}

// 연결 관리자 종료
func (cm *ConnectionManager) Stop() {
	loghandle.Info("연결 관리자 종료 시작")
	
	close(cm.shutdownChan)
	
	// 모든 연결 종료
	cm.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*Connection); ok {
			conn.Close()
		}
		return true
	})
	
	cm.wg.Wait()
	loghandle.Info("연결 관리자 종료 완료")
}

// 새 연결 추가
func (cm *ConnectionManager) AddConnection(netConn net.Conn) (*Connection, error) {
	// 최대 연결 수 체크
	if int(atomic.LoadInt32(&cm.currentCount)) >= cm.maxConnections {
		netConn.Close()
		return nil, fmt.Errorf("최대 연결 수 초과: %d", cm.maxConnections)
	}
	
	// 새 연결 ID 생성
	id := atomic.AddUint64(&cm.idGenerator, 1)
	
	// 연결 객체 생성
	conn := NewConnection(id, netConn, cm)
	
	// 연결 저장
	cm.connections.Store(id, conn)
	atomic.AddInt32(&cm.currentCount, 1)
	
	// 연결 시작
	conn.Start()
	
	// 연결 이벤트 전송
	select {
	case cm.connectChan <- conn:
	default:
		loghandle.Warn("연결 이벤트 채널이 가득 참")
	}
	
	loghandle.Info("새 연결 추가: ID=%d, Remote=%s", id, netConn.RemoteAddr())
	return conn, nil
}

// 연결 제거
func (cm *ConnectionManager) removeConnection(id uint64) {
	if _, loaded := cm.connections.LoadAndDelete(id); loaded {
		atomic.AddInt32(&cm.currentCount, -1)
		
		// 연결 해제 이벤트 전송
		select {
		case cm.disconnectChan <- id:
		default:
			loghandle.Warn("연결 해제 이벤트 채널이 가득 찬")
		}
		
		loghandle.Info("연결 제거: ID=%d", id)
	}
}

// 연결 조회
func (cm *ConnectionManager) GetConnection(id uint64) (*Connection, bool) {
	if value, ok := cm.connections.Load(id); ok {
		return value.(*Connection), true
	}
	return nil, false
}

// 플레이어 ID로 연결 조회
func (cm *ConnectionManager) GetConnectionByPlayerID(playerID uint64) (*Connection, bool) {
	var result *Connection
	found := false
	
	cm.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*Connection); ok {
			if conn.PlayerSession != nil && conn.PlayerSession.PlayerID == playerID {
				result = conn
				found = true
				return false // 찾았으므로 반복 중단
			}
		}
		return true // 계속 검색
	})
	
	return result, found
}

// 모든 연결 조회
func (cm *ConnectionManager) GetAllConnections() []*Connection {
	var connections []*Connection
	
	cm.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*Connection); ok {
			connections = append(connections, conn)
		}
		return true
	})
	
	return connections
}

// 패킷 처리를 위해 이벤트 전송
func (cm *ConnectionManager) submitPacketForProcessing(conn *Connection, packet protocol.Packet) error {
	event := &PacketEvent{
		Connection: conn,
		Packet:     packet,
		Timestamp:  time.Now(),
	}
	
	select {
	case cm.packetChan <- event:
		return nil
	default:
		return fmt.Errorf("패킷 처리 채널이 가득 참")
	}
}

// 패킷 이벤트 채널 조회 (패킷 프로세서에서 사용)
func (cm *ConnectionManager) GetPacketChannel() <-chan *PacketEvent {
	return cm.packetChan
}

// 연결 이벤트 루프
func (cm *ConnectionManager) connectionEventLoop() {
	defer cm.wg.Done()
	
	for {
		select {
		case conn := <-cm.connectChan:
			loghandle.Debug("새 연결 이벤트 처리: ID=%d", conn.ID)
			// 여기서 연결 관련 추가 처리 가능
			
		case <-cm.shutdownChan:
			return
		}
	}
}

// 연결 해제 이벤트 루프
func (cm *ConnectionManager) disconnectionEventLoop() {
	defer cm.wg.Done()
	
	for {
		select {
		case id := <-cm.disconnectChan:
			loghandle.Debug("연결 해제 이벤트 처리: ID=%d", id)
			// 여기서 연결 해제 관련 추가 처리 가능
			
		case <-cm.shutdownChan:
			return
		}
	}
}

// 정리 루프 (비활성 연결 제거)
func (cm *ConnectionManager) cleanupLoop() {
	defer cm.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cm.cleanupInactiveConnections()
			
		case <-cm.shutdownChan:
			return
		}
	}
}

// 비활성 연결 정리
func (cm *ConnectionManager) cleanupInactiveConnections() {
	timeout := 5 * time.Minute
	now := time.Now()
	toRemove := []uint64{}
	
	cm.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*Connection); ok {
			if now.Sub(conn.LastActivity) > timeout {
				toRemove = append(toRemove, conn.ID)
			}
		}
		return true
	})
	
	for _, id := range toRemove {
		if conn, ok := cm.GetConnection(id); ok {
			loghandle.Info("비활성 연결 제거: ID=%d", id)
			conn.Close()
		}
	}
}

// 통계 정보
func (cm *ConnectionManager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"current_connections": atomic.LoadInt32(&cm.currentCount),
		"max_connections":     cm.maxConnections,
		"total_created":       atomic.LoadUint64(&cm.idGenerator),
	}
}

// 브로드캐스트 (모든 연결에게 패킷 전송)
func (cm *ConnectionManager) Broadcast(packet protocol.Packet) {
	cm.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*Connection); ok {
			if conn.IsConnected() {
				go conn.SendPacket(packet)
			}
		}
		return true
	})
}

// 특정 게임의 플레이어들에게 브로드캐스트
func (cm *ConnectionManager) BroadcastToGame(gameID protocol.GameID, packet protocol.Packet) {
	cm.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*Connection); ok {
			if conn.IsConnected() && conn.CurrentGame == gameID {
				go conn.SendPacket(packet)
			}
		}
		return true
	})
}