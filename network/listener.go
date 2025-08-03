package network

import (
	"go_server_framework/loghandle"
	"net"
	"sync"
)

// TCP 리스너
type Listener struct {
	listener    net.Listener
	connManager *ConnectionManager
	
	// 제어
	shutdownChan chan struct{}
	wg           sync.WaitGroup
}

func NewListener(address string, connManager *ConnectionManager) (*Listener, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	
	return &Listener{
		listener:     listener,
		connManager:  connManager,
		shutdownChan: make(chan struct{}),
	}, nil
}

// 리스너 시작
func (l *Listener) Start() {
	loghandle.Info("TCP 리스너 시작: %s", l.listener.Addr())
	
	l.wg.Add(1)
	go l.acceptLoop()
}

// 리스너 종료
func (l *Listener) Stop() {
	loghandle.Info("TCP 리스너 종료")
	
	close(l.shutdownChan)
	
	if l.listener != nil {
		l.listener.Close()
	}
	
	l.wg.Wait()
	loghandle.Info("TCP 리스너 종료 완료")
}

// Accept 루프
func (l *Listener) acceptLoop() {
	defer l.wg.Done()
	
	for {
		select {
		case <-l.shutdownChan:
			return
		default:
			// 새 연결 수락
			conn, err := l.listener.Accept()
			if err != nil {
				select {
				case <-l.shutdownChan:
					return // 종료 중이면 에러 무시
				default:
					loghandle.Error("Accept 실패: %v", err)
					continue
				}
			}
			
			// 연결 관리자에 추가
			_, err = l.connManager.AddConnection(conn)
			if err != nil {
				loghandle.Error("연결 추가 실패: %v", err)
				conn.Close()
			}
		}
	}
}

// 리스너 주소 조회
func (l *Listener) Addr() net.Addr {
	return l.listener.Addr()
}