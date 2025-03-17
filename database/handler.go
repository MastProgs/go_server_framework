package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go_server_framework/loghandle"
)

// Handler는 데이터베이스 작업을 위한 핸들러입니다
type Handler struct {
	db *sql.DB
}

// NewHandler는 새로운 데이터베이스 핸들러를 생성합니다
func NewHandler() (*Handler, error) {
	db := GetDB()
	if db == nil {
		return &Handler{db: nil}, errors.New("데이터베이스 연결이 없습니다")
	}
	return &Handler{db: db}, nil
}

// IsConnected는 핸들러가 유효한 데이터베이스 연결을 가지고 있는지 확인합니다
func (h *Handler) IsConnected() bool {
	return h.db != nil
}

// Exec는 SQL 쿼리를 실행하고 결과를 반환합니다
func (h *Handler) Exec(query string, args ...interface{}) (sql.Result, error) {
	if !h.IsConnected() {
		return nil, errors.New("데이터베이스 연결이 없습니다")
	}

	startTime := time.Now()
	result, err := h.db.Exec(query, args...)
	duration := time.Since(startTime)

	if err != nil {
		loghandle.Error("SQL 쿼리 실행 실패: %v, 쿼리: %s, 인자: %v", err, query, args)
		return nil, err
	}

	loghandle.Debug("SQL 쿼리 실행 성공: 쿼리: %s, 소요 시간: %v", query, duration)
	return result, nil
}

// ExecContext는 컨텍스트와 함께 SQL 쿼리를 실행하고 결과를 반환합니다
func (h *Handler) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if !h.IsConnected() {
		return nil, errors.New("데이터베이스 연결이 없습니다")
	}

	startTime := time.Now()
	result, err := h.db.ExecContext(ctx, query, args...)
	duration := time.Since(startTime)

	if err != nil {
		loghandle.Error("SQL 쿼리 실행 실패 (컨텍스트): %v, 쿼리: %s, 인자: %v", err, query, args)
		return nil, err
	}

	loghandle.Debug("SQL 쿼리 실행 성공 (컨텍스트): 쿼리: %s, 소요 시간: %v", query, duration)
	return result, nil
}

// Query는 SQL 쿼리를 실행하고 결과 행을 반환합니다
func (h *Handler) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if !h.IsConnected() {
		return nil, errors.New("데이터베이스 연결이 없습니다")
	}

	startTime := time.Now()
	rows, err := h.db.Query(query, args...)
	duration := time.Since(startTime)

	if err != nil {
		loghandle.Error("SQL 쿼리 조회 실패: %v, 쿼리: %s, 인자: %v", err, query, args)
		return nil, err
	}

	loghandle.Debug("SQL 쿼리 조회 성공: 쿼리: %s, 소요 시간: %v", query, duration)
	return rows, nil
}

// QueryContext는 컨텍스트와 함께 SQL 쿼리를 실행하고 결과 행을 반환합니다
func (h *Handler) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if !h.IsConnected() {
		return nil, errors.New("데이터베이스 연결이 없습니다")
	}

	startTime := time.Now()
	rows, err := h.db.QueryContext(ctx, query, args...)
	duration := time.Since(startTime)

	if err != nil {
		loghandle.Error("SQL 쿼리 조회 실패 (컨텍스트): %v, 쿼리: %s, 인자: %v", err, query, args)
		return nil, err
	}

	loghandle.Debug("SQL 쿼리 조회 성공 (컨텍스트): 쿼리: %s, 소요 시간: %v", query, duration)
	return rows, nil
}

// QueryRow는 SQL 쿼리를 실행하고 단일 행을 반환합니다
func (h *Handler) QueryRow(query string, args ...interface{}) *sql.Row {
	if !h.IsConnected() {
		// 데이터베이스 연결이 없는 경우 빈 Row 반환
		// sql.Row는 Scan 호출 시 오류를 반환하므로 안전함
		loghandle.Warn("데이터베이스 연결 없이 QueryRow 호출: %s", query)
		return &sql.Row{}
	}

	startTime := time.Now()
	row := h.db.QueryRow(query, args...)
	duration := time.Since(startTime)

	loghandle.Debug("SQL 단일 행 조회: 쿼리: %s, 소요 시간: %v", query, duration)
	return row
}

// QueryRowContext는 컨텍스트와 함께 SQL 쿼리를 실행하고 단일 행을 반환합니다
func (h *Handler) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if !h.IsConnected() {
		// 데이터베이스 연결이 없는 경우 빈 Row 반환
		loghandle.Warn("데이터베이스 연결 없이 QueryRowContext 호출: %s", query)
		return &sql.Row{}
	}

	startTime := time.Now()
	row := h.db.QueryRowContext(ctx, query, args...)
	duration := time.Since(startTime)

	loghandle.Debug("SQL 단일 행 조회 (컨텍스트): 쿼리: %s, 소요 시간: %v", query, duration)
	return row
}

// Begin은 새로운 트랜잭션을 시작합니다
func (h *Handler) Begin() (*Transaction, error) {
	if !h.IsConnected() {
		return nil, errors.New("데이터베이스 연결이 없습니다")
	}

	tx, err := h.db.Begin()
	if err != nil {
		loghandle.Error("트랜잭션 시작 실패: %v", err)
		return nil, err
	}

	loghandle.Debug("트랜잭션 시작")
	return &Transaction{tx: tx}, nil
}

// BeginTx는 컨텍스트와 함께 새로운 트랜잭션을 시작합니다
func (h *Handler) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Transaction, error) {
	if !h.IsConnected() {
		return nil, errors.New("데이터베이스 연결이 없습니다")
	}

	tx, err := h.db.BeginTx(ctx, opts)
	if err != nil {
		loghandle.Error("트랜잭션 시작 실패 (컨텍스트): %v", err)
		return nil, err
	}

	loghandle.Debug("트랜잭션 시작 (컨텍스트)")
	return &Transaction{tx: tx}, nil
}

// Prepare는 SQL 문을 준비합니다
func (h *Handler) Prepare(query string) (*sql.Stmt, error) {
	if !h.IsConnected() {
		return nil, errors.New("데이터베이스 연결이 없습니다")
	}

	stmt, err := h.db.Prepare(query)
	if err != nil {
		loghandle.Error("SQL 문 준비 실패: %v, 쿼리: %s", err, query)
		return nil, err
	}

	loghandle.Debug("SQL 문 준비 성공: 쿼리: %s", query)
	return stmt, nil
}

// PrepareContext는 컨텍스트와 함께 SQL 문을 준비합니다
func (h *Handler) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if !h.IsConnected() {
		return nil, errors.New("데이터베이스 연결이 없습니다")
	}

	stmt, err := h.db.PrepareContext(ctx, query)
	if err != nil {
		loghandle.Error("SQL 문 준비 실패 (컨텍스트): %v, 쿼리: %s", err, query)
		return nil, err
	}

	loghandle.Debug("SQL 문 준비 성공 (컨텍스트): 쿼리: %s", query)
	return stmt, nil
}

// Transaction은 데이터베이스 트랜잭션을 나타냅니다
type Transaction struct {
	tx *sql.Tx
}

// Commit은 트랜잭션을 커밋합니다
func (t *Transaction) Commit() error {
	if err := t.tx.Commit(); err != nil {
		loghandle.Error("트랜잭션 커밋 실패: %v", err)
		return err
	}

	loghandle.Debug("트랜잭션 커밋 성공")
	return nil
}

// Rollback은 트랜잭션을 롤백합니다
func (t *Transaction) Rollback() error {
	if err := t.tx.Rollback(); err != nil {
		loghandle.Error("트랜잭션 롤백 실패: %v", err)
		return err
	}

	loghandle.Debug("트랜잭션 롤백 성공")
	return nil
}

// Exec는 트랜잭션 내에서 SQL 쿼리를 실행합니다
func (t *Transaction) Exec(query string, args ...interface{}) (sql.Result, error) {
	startTime := time.Now()
	result, err := t.tx.Exec(query, args...)
	duration := time.Since(startTime)

	if err != nil {
		loghandle.Error("트랜잭션 SQL 쿼리 실행 실패: %v, 쿼리: %s, 인자: %v", err, query, args)
		return nil, err
	}

	loghandle.Debug("트랜잭션 SQL 쿼리 실행 성공: 쿼리: %s, 소요 시간: %v", query, duration)
	return result, nil
}

// ExecContext는 컨텍스트와 함께 트랜잭션 내에서 SQL 쿼리를 실행합니다
func (t *Transaction) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	startTime := time.Now()
	result, err := t.tx.ExecContext(ctx, query, args...)
	duration := time.Since(startTime)

	if err != nil {
		loghandle.Error("트랜잭션 SQL 쿼리 실행 실패 (컨텍스트): %v, 쿼리: %s, 인자: %v", err, query, args)
		return nil, err
	}

	loghandle.Debug("트랜잭션 SQL 쿼리 실행 성공 (컨텍스트): 쿼리: %s, 소요 시간: %v", query, duration)
	return result, nil
}

// Query는 트랜잭션 내에서 SQL 쿼리를 실행하고 결과 행을 반환합니다
func (t *Transaction) Query(query string, args ...interface{}) (*sql.Rows, error) {
	startTime := time.Now()
	rows, err := t.tx.Query(query, args...)
	duration := time.Since(startTime)

	if err != nil {
		loghandle.Error("트랜잭션 SQL 쿼리 조회 실패: %v, 쿼리: %s, 인자: %v", err, query, args)
		return nil, err
	}

	loghandle.Debug("트랜잭션 SQL 쿼리 조회 성공: 쿼리: %s, 소요 시간: %v", query, duration)
	return rows, nil
}

// QueryContext는 컨텍스트와 함께 트랜잭션 내에서 SQL 쿼리를 실행하고 결과 행을 반환합니다
func (t *Transaction) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	startTime := time.Now()
	rows, err := t.tx.QueryContext(ctx, query, args...)
	duration := time.Since(startTime)

	if err != nil {
		loghandle.Error("트랜잭션 SQL 쿼리 조회 실패 (컨텍스트): %v, 쿼리: %s, 인자: %v", err, query, args)
		return nil, err
	}

	loghandle.Debug("트랜잭션 SQL 쿼리 조회 성공 (컨텍스트): 쿼리: %s, 소요 시간: %v", query, duration)
	return rows, nil
}

// QueryRow는 트랜잭션 내에서 SQL 쿼리를 실행하고 단일 행을 반환합니다
func (t *Transaction) QueryRow(query string, args ...interface{}) *sql.Row {
	startTime := time.Now()
	row := t.tx.QueryRow(query, args...)
	duration := time.Since(startTime)

	loghandle.Debug("트랜잭션 SQL 단일 행 조회: 쿼리: %s, 소요 시간: %v", query, duration)
	return row
}

// QueryRowContext는 컨텍스트와 함께 트랜잭션 내에서 SQL 쿼리를 실행하고 단일 행을 반환합니다
func (t *Transaction) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	startTime := time.Now()
	row := t.tx.QueryRowContext(ctx, query, args...)
	duration := time.Since(startTime)

	loghandle.Debug("트랜잭션 SQL 단일 행 조회 (컨텍스트): 쿼리: %s, 소요 시간: %v", query, duration)
	return row
}

// Prepare는 트랜잭션 내에서 SQL 문을 준비합니다
func (t *Transaction) Prepare(query string) (*sql.Stmt, error) {
	stmt, err := t.tx.Prepare(query)
	if err != nil {
		loghandle.Error("트랜잭션 SQL 문 준비 실패: %v, 쿼리: %s", err, query)
		return nil, err
	}

	loghandle.Debug("트랜잭션 SQL 문 준비 성공: 쿼리: %s", query)
	return stmt, nil
}

// PrepareContext는 컨텍스트와 함께 트랜잭션 내에서 SQL 문을 준비합니다
func (t *Transaction) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	stmt, err := t.tx.PrepareContext(ctx, query)
	if err != nil {
		loghandle.Error("트랜잭션 SQL 문 준비 실패 (컨텍스트): %v, 쿼리: %s", err, query)
		return nil, err
	}

	loghandle.Debug("트랜잭션 SQL 문 준비 성공 (컨텍스트): 쿼리: %s", query)
	return stmt, nil
}

// ExecuteWithTransaction은 트랜잭션 내에서 함수를 실행합니다
func ExecuteWithTransaction(fn func(*Transaction) error) error {
	handler, err := NewHandler()
	if err != nil {
		return fmt.Errorf("데이터베이스 핸들러 생성 실패: %w", err)
	}

	tx, err := handler.Begin()
	if err != nil {
		return fmt.Errorf("트랜잭션 시작 실패: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			// 패닉 발생 시 롤백
			_ = tx.Rollback()
			panic(p) // 패닉 다시 발생
		}
	}()

	if err := fn(tx); err != nil {
		// 오류 발생 시 롤백
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("트랜잭션 실행 실패: %v, 롤백 실패: %v", err, rbErr)
		}
		return err
	}

	// 성공 시 커밋
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("트랜잭션 커밋 실패: %w", err)
	}

	return nil
}
