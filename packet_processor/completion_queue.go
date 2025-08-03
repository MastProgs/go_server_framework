package packet_processor

import (
	workerpool "go_server_framework/core"
	"go_server_framework/loghandle"
	"go_server_framework/network"
	"go_server_framework/protocol"
	"sync"
	"time"
)

// 완료 이벤트 타입 (IOCP 스타일)
type CompletionType int

const (
	COMPLETION_PACKET_RECEIVED CompletionType = iota
	COMPLETION_PACKET_SENT
	COMPLETION_CONNECTION_CLOSED
	COMPLETION_ERROR
)

// 완료 이벤트 (IOCP의 OVERLAPPED_ENTRY와 유사)
type CompletionEntry struct {
	Type       CompletionType
	Connection *network.Connection
	Packet     protocol.Packet
	Error      error
	Timestamp  time.Time
	Context    interface{} // 추가 컨텍스트 데이터
}

// 완료 큐 (IOCP의 I/O Completion Port와 유사)
type CompletionQueue struct {
	queue      chan *CompletionEntry
	maxWorkers int
	workerPool *workerpool.WorkerPool
	router     *GameRouter

	// 통계
	processedCount uint64
	errorCount     uint64

	// 제어
	shutdownChan chan struct{}
	wg           sync.WaitGroup
}

func NewCompletionQueue(queueSize int, maxWorkers int, router *GameRouter) *CompletionQueue {
	return &CompletionQueue{
		queue:        make(chan *CompletionEntry, queueSize),
		maxWorkers:   maxWorkers,
		workerPool:   workerpool.NewWorkerPool(),
		router:       router,
		shutdownChan: make(chan struct{}),
	}
}

// 완료 큐 시작 (IOCP 초기화와 유사)
func (cq *CompletionQueue) Start() {
	loghandle.Info("완료 큐 시작 (워커: %d개)", cq.maxWorkers)

	// 워커 풀 시작
	cq.workerPool.Start()

	// 완료 이벤트 처리 고루틴들 시작
	for i := 0; i < cq.maxWorkers; i++ {
		cq.wg.Add(1)
		go cq.completionWorker(i)
	}

	loghandle.Info("완료 큐 시작 완료")
}

// 완료 큐 종료
func (cq *CompletionQueue) Stop() {
	loghandle.Info("완료 큐 종료 시작")

	close(cq.shutdownChan)
	cq.workerPool.Stop()

	cq.wg.Wait()

	loghandle.Info("완료 큐 종료 완료")
}

// 완료 이벤트 추가 (IOCP의 PostQueuedCompletionStatus와 유사)
func (cq *CompletionQueue) PostCompletion(entry *CompletionEntry) error {
	select {
	case cq.queue <- entry:
		return nil
	case <-cq.shutdownChan:
		return ErrQueueClosed
	default:
		return ErrQueueFull
	}
}

// 패킷 수신 완료 이벤트 생성
func (cq *CompletionQueue) PostPacketReceived(conn *network.Connection, packet protocol.Packet) error {
	entry := &CompletionEntry{
		Type:       COMPLETION_PACKET_RECEIVED,
		Connection: conn,
		Packet:     packet,
		Timestamp:  time.Now(),
	}

	return cq.PostCompletion(entry)
}

// 연결 종료 완료 이벤트 생성
func (cq *CompletionQueue) PostConnectionClosed(conn *network.Connection) error {
	entry := &CompletionEntry{
		Type:       COMPLETION_CONNECTION_CLOSED,
		Connection: conn,
		Timestamp:  time.Now(),
	}

	return cq.PostCompletion(entry)
}

// 에러 완료 이벤트 생성
func (cq *CompletionQueue) PostError(conn *network.Connection, err error) error {
	entry := &CompletionEntry{
		Type:       COMPLETION_ERROR,
		Connection: conn,
		Error:      err,
		Timestamp:  time.Now(),
	}

	return cq.PostCompletion(entry)
}

// 완료 이벤트 워커 (IOCP의 GetQueuedCompletionStatus 루프와 유사)
func (cq *CompletionQueue) completionWorker(workerID int) {
	defer cq.wg.Done()

	loghandle.Debug("완료 큐 워커 %d 시작", workerID)

	for {
		select {
		case entry := <-cq.queue:
			cq.handleCompletionEntry(entry, workerID)

		case <-cq.shutdownChan:
			loghandle.Debug("완료 큐 워커 %d 종료", workerID)
			return
		}
	}
}

// 완료 이벤트 처리
func (cq *CompletionQueue) handleCompletionEntry(entry *CompletionEntry, workerID int) {
	defer func() {
		if r := recover(); r != nil {
			loghandle.Error("완료 이벤트 처리 중 패닉 발생 (워커 %d): %v", workerID, r)
			cq.errorCount++
		}
	}()

	switch entry.Type {
	case COMPLETION_PACKET_RECEIVED:
		cq.handlePacketReceived(entry)

	case COMPLETION_CONNECTION_CLOSED:
		cq.handleConnectionClosed(entry)

	case COMPLETION_ERROR:
		cq.handleError(entry)

	default:
		loghandle.Warn("알 수 없는 완료 이벤트 타입: %d", entry.Type)
	}

	cq.processedCount++
}

// 패킷 수신 처리
func (cq *CompletionQueue) handlePacketReceived(entry *CompletionEntry) {
	// 연결 활성화 시간 업데이트
	entry.Connection.UpdateActivity()

	// 게임 라우터로 패킷 전달
	if err := cq.router.RoutePacket(entry.Connection, entry.Packet); err != nil {
		loghandle.Error("패킷 라우팅 실패: %v", err)
		cq.errorCount++

		// 에러 응답 전송
		cq.sendErrorResponse(entry.Connection, err)
	}
}

// 연결 종료 처리
func (cq *CompletionQueue) handleConnectionClosed(entry *CompletionEntry) {
	loghandle.Debug("연결 종료 처리: ID=%d", entry.Connection.ID)

	// 게임 라우터에 연결 종료 알림
	cq.router.HandleConnectionClosed(entry.Connection)
}

// 에러 처리
func (cq *CompletionQueue) handleError(entry *CompletionEntry) {
	loghandle.Error("연결 에러 처리: ID=%d, Error=%v", entry.Connection.ID, entry.Error)

	// 연결 종료
	entry.Connection.Close()
}

// 에러 응답 전송
func (cq *CompletionQueue) sendErrorResponse(conn *network.Connection, err error) {
	// TODO: 에러 응답 패킷 생성 및 전송
	loghandle.Debug("에러 응답 전송: %v", err)
}

// 통계 조회
func (cq *CompletionQueue) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"queue_size":      len(cq.queue),
		"max_workers":     cq.maxWorkers,
		"processed_count": cq.processedCount,
		"error_count":     cq.errorCount,
	}
}

// 큐 크기 조회
func (cq *CompletionQueue) GetQueueSize() int {
	return len(cq.queue)
}

// 큐가 가득 찬지 확인
func (cq *CompletionQueue) IsFull() bool {
	return len(cq.queue) == cap(cq.queue)
}

// 워커 풀에 작업 제출 (추가 비동기 처리용)
func (cq *CompletionQueue) SubmitWork(work func()) {
	cq.workerPool.Submit(work)
}
