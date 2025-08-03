package network

import (
	"go_server_framework/loghandle"
	"go_server_framework/protocol"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// 연결 상태
type ConnectionState int32

const (
	STATE_CONNECTING ConnectionState = iota
	STATE_CONNECTED
	STATE_AUTHENTICATED
	STATE_DISCONNECTING
	STATE_DISCONNECTED
)

// 플레이어 세션 정보
type PlayerSession struct {
	PlayerID     uint64
	Username     string
	LoginTime    time.Time
	LastActivity time.Time
	GameAccess   []protocol.GameID
}

// 개별 TCP 연결
type Connection struct {
	ID           uint64
	conn         net.Conn
	state        int32 // atomic 사용
	sendChan     chan []byte
	receiveBuf   []byte
	
	// 게임 관련 컨텍스트
	PlayerSession *PlayerSession
	CurrentGame   protocol.GameID
	
	// 성능 관련
	LastActivity time.Time
	PacketCount  uint64
	BytesSent    uint64
	BytesRecv    uint64
	
	// 제어
	closeChan    chan struct{}
	closeOnce    sync.Once
	manager      *ConnectionManager
}

func NewConnection(id uint64, conn net.Conn, manager *ConnectionManager) *Connection {
	return &Connection{
		ID:           id,
		conn:         conn,
		state:        int32(STATE_CONNECTING),
		sendChan:     make(chan []byte, 256),
		receiveBuf:   make([]byte, 4096),
		LastActivity: time.Now(),
		closeChan:    make(chan struct{}),
		manager:      manager,
	}
}

// 연결 상태 관리
func (c *Connection) GetState() ConnectionState {
	return ConnectionState(atomic.LoadInt32(&c.state))
}

func (c *Connection) SetState(state ConnectionState) {
	atomic.StoreInt32(&c.state, int32(state))
}

func (c *Connection) IsConnected() bool {
	state := c.GetState()
	return state == STATE_CONNECTED || state == STATE_AUTHENTICATED
}

// 연결 시작 (읽기/쓰기 고루틴 시작)
func (c *Connection) Start() {
	c.SetState(STATE_CONNECTED)
	go c.readLoop()
	go c.writeLoop()
}

// 읽기 루프 (수신 패킷 처리)
func (c *Connection) readLoop() {
	defer c.Close()
	
	for {
		select {
		case <-c.closeChan:
			return
		default:
			// 논블로킹 읽기 시도
			c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, err := c.conn.Read(c.receiveBuf)
			
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // 타임아웃은 정상
				}
				return // 실제 에러
			}
			
			if n > 0 {
				c.LastActivity = time.Now()
				atomic.AddUint64(&c.BytesRecv, uint64(n))
				atomic.AddUint64(&c.PacketCount, 1)
				
				// 패킷 파싱 및 처리
				if err := c.processReceivedData(c.receiveBuf[:n]); err != nil {
					loghandle.Error("패킷 처리 실패: %v", err)
					return
				}
			}
		}
	}
}

// 쓰기 루프 (송신 패킷 처리)
func (c *Connection) writeLoop() {
	defer c.Close()
	
	for {
		select {
		case data := <-c.sendChan:
			_, err := c.conn.Write(data)
			if err != nil {
				return
			}
			atomic.AddUint64(&c.BytesSent, uint64(len(data)))
			
		case <-c.closeChan:
			return
		}
	}
}

// 수신된 데이터 처리
func (c *Connection) processReceivedData(data []byte) error {
	// 패킷 크기가 충분한지 확인
	if len(data) < protocol.HeaderSize {
		return protocol.ErrInvalidPacketSize
	}
	
	// 헤더 파싱
	header, err := protocol.DeserializeHeader(data)
	if err != nil {
		return err
	}
	
	// 패킷 크기 검증
	if len(data) < int(header.Length) {
		return protocol.ErrInvalidPacketSize
	}
	
	// 완전한 패킷 역직렬화
	packet, err := protocol.DeserializePacket(data[:header.Length])
	if err != nil {
		return err
	}
	
	// 완료 큐에 전송
	return c.manager.submitPacketForProcessing(c, packet)
}

// 패킷 전송
func (c *Connection) SendPacket(packet protocol.Packet) error {
	if !c.IsConnected() {
		return net.ErrClosed
	}
	
	data, err := protocol.SerializePacket(packet)
	if err != nil {
		return err
	}
	
	select {
	case c.sendChan <- data:
		return nil
	case <-c.closeChan:
		return net.ErrClosed
	default:
		// 채널이 가득 찬 경우 연결 끊기
		go c.Close()
		return net.ErrClosed
	}
}

// 연결 종료
func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		c.SetState(STATE_DISCONNECTING)
		close(c.closeChan)
		
		if c.conn != nil {
			c.conn.Close()
		}
		
		c.SetState(STATE_DISCONNECTED)
		
		// 매니저에게 연결 종료 알림
		c.manager.removeConnection(c.ID)
	})
}

// 연결 정보 조회
func (c *Connection) GetInfo() map[string]interface{} {
	return map[string]interface{}{
		"id":            c.ID,
		"state":         c.GetState(),
		"remote_addr":   c.conn.RemoteAddr().String(),
		"last_activity": c.LastActivity,
		"packet_count":  atomic.LoadUint64(&c.PacketCount),
		"bytes_sent":    atomic.LoadUint64(&c.BytesSent),
		"bytes_recv":    atomic.LoadUint64(&c.BytesRecv),
		"player_id":     c.getPlayerID(),
		"current_game":  c.CurrentGame,
	}
}

func (c *Connection) getPlayerID() uint64 {
	if c.PlayerSession != nil {
		return c.PlayerSession.PlayerID
	}
	return 0
}

// 세션 관리
func (c *Connection) SetPlayerSession(session *PlayerSession) {
	c.PlayerSession = session
	c.SetState(STATE_AUTHENTICATED)
}

func (c *Connection) ClearPlayerSession() {
	c.PlayerSession = nil
	if c.GetState() == STATE_AUTHENTICATED {
		c.SetState(STATE_CONNECTED)
	}
}

// 활성화 시간 업데이트
func (c *Connection) UpdateActivity() {
	c.LastActivity = time.Now()
}