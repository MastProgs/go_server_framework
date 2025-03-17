package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"go_server_framework/loghandle"
)

// Model은 데이터베이스 모델의 기본 인터페이스입니다
type Model interface {
	TableName() string
	PrimaryKey() string
}

// Repository는 모델에 대한 CRUD 작업을 제공합니다
type Repository struct {
	handler *Handler
}

// NewRepository는 새로운 리포지토리를 생성합니다
func NewRepository() (*Repository, error) {
	handler, err := NewHandler()
	if err != nil {
		loghandle.Warn("데이터베이스 핸들러 생성 실패: %v - 제한된 기능으로 계속합니다", err)
		// 오류가 있어도 핸들러는 생성됨 (db가 nil인 상태)
	}
	return &Repository{handler: handler}, nil
}

// IsConnected는 리포지토리가 유효한 데이터베이스 연결을 가지고 있는지 확인합니다
func (r *Repository) IsConnected() bool {
	return r.handler != nil && r.handler.IsConnected()
}

// Create는 모델을 데이터베이스에 삽입합니다
func (r *Repository) Create(model Model) (int64, error) {
	if !r.IsConnected() {
		return 0, errors.New("데이터베이스 연결이 없습니다")
	}

	fields, values, placeholders := extractFieldsAndValues(model)
	if len(fields) == 0 {
		return 0, errors.New("모델에 필드가 없습니다")
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		model.TableName(),
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "))

	result, err := r.handler.Exec(query, values...)
	if err != nil {
		return 0, fmt.Errorf("모델 생성 실패: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("마지막 삽입 ID 가져오기 실패: %w", err)
	}

	return id, nil
}

// CreateContext는 컨텍스트와 함께 모델을 데이터베이스에 삽입합니다
func (r *Repository) CreateContext(ctx context.Context, model Model) (int64, error) {
	if !r.IsConnected() {
		return 0, errors.New("데이터베이스 연결이 없습니다")
	}

	fields, values, placeholders := extractFieldsAndValues(model)
	if len(fields) == 0 {
		return 0, errors.New("모델에 필드가 없습니다")
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		model.TableName(),
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "))

	result, err := r.handler.ExecContext(ctx, query, values...)
	if err != nil {
		return 0, fmt.Errorf("모델 생성 실패 (컨텍스트): %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("마지막 삽입 ID 가져오기 실패: %w", err)
	}

	return id, nil
}

// FindByID는 ID로 모델을 조회합니다
func (r *Repository) FindByID(model Model, id interface{}) error {
	if !r.IsConnected() {
		return errors.New("데이터베이스 연결이 없습니다")
	}

	fields := extractFields(model)
	if len(fields) == 0 {
		return errors.New("모델에 필드가 없습니다")
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?",
		strings.Join(fields, ", "),
		model.TableName(),
		model.PrimaryKey())

	row := r.handler.QueryRow(query, id)
	if err := scanModel(row, model); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("ID가 %v인 모델을 찾을 수 없습니다", id)
		}
		return fmt.Errorf("모델 조회 실패: %w", err)
	}

	return nil
}

// FindByIDContext는 컨텍스트와 함께 ID로 모델을 조회합니다
func (r *Repository) FindByIDContext(ctx context.Context, model Model, id interface{}) error {
	if !r.IsConnected() {
		return errors.New("데이터베이스 연결이 없습니다")
	}

	fields := extractFields(model)
	if len(fields) == 0 {
		return errors.New("모델에 필드가 없습니다")
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?",
		strings.Join(fields, ", "),
		model.TableName(),
		model.PrimaryKey())

	row := r.handler.QueryRowContext(ctx, query, id)
	if err := scanModel(row, model); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("ID가 %v인 모델을 찾을 수 없습니다", id)
		}
		return fmt.Errorf("모델 조회 실패 (컨텍스트): %w", err)
	}

	return nil
}

// FindAll은 모든 모델을 조회합니다
func (r *Repository) FindAll(model Model, dest interface{}) error {
	if !r.IsConnected() {
		return errors.New("데이터베이스 연결이 없습니다")
	}

	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return errors.New("dest는 슬라이스 포인터여야 합니다")
	}

	fields := extractFields(model)
	if len(fields) == 0 {
		return errors.New("모델에 필드가 없습니다")
	}

	query := fmt.Sprintf("SELECT %s FROM %s",
		strings.Join(fields, ", "),
		model.TableName())

	rows, err := r.handler.Query(query)
	if err != nil {
		return fmt.Errorf("모델 조회 실패: %w", err)
	}
	defer rows.Close()

	sliceValue := destValue.Elem()
	elemType := sliceValue.Type().Elem()

	for rows.Next() {
		newElem := reflect.New(elemType).Elem()
		modelInstance := newElem.Addr().Interface().(Model)

		if err := scanModel(rows, modelInstance); err != nil {
			return fmt.Errorf("모델 스캔 실패: %w", err)
		}

		sliceValue.Set(reflect.Append(sliceValue, newElem))
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("행 반복 중 오류 발생: %w", err)
	}

	return nil
}

// FindAllContext는 컨텍스트와 함께 모든 모델을 조회합니다
func (r *Repository) FindAllContext(ctx context.Context, model Model, dest interface{}) error {
	if !r.IsConnected() {
		return errors.New("데이터베이스 연결이 없습니다")
	}

	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return errors.New("dest는 슬라이스 포인터여야 합니다")
	}

	fields := extractFields(model)
	if len(fields) == 0 {
		return errors.New("모델에 필드가 없습니다")
	}

	query := fmt.Sprintf("SELECT %s FROM %s",
		strings.Join(fields, ", "),
		model.TableName())

	rows, err := r.handler.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("모델 조회 실패 (컨텍스트): %w", err)
	}
	defer rows.Close()

	sliceValue := destValue.Elem()
	elemType := sliceValue.Type().Elem()

	for rows.Next() {
		newElem := reflect.New(elemType).Elem()
		modelInstance := newElem.Addr().Interface().(Model)

		if err := scanModel(rows, modelInstance); err != nil {
			return fmt.Errorf("모델 스캔 실패: %w", err)
		}

		sliceValue.Set(reflect.Append(sliceValue, newElem))
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("행 반복 중 오류 발생: %w", err)
	}

	return nil
}

// Update는 모델을 업데이트합니다
func (r *Repository) Update(model Model) (int64, error) {
	if !r.IsConnected() {
		return 0, errors.New("데이터베이스 연결이 없습니다")
	}

	fields, values, _ := extractFieldsAndValues(model)
	if len(fields) == 0 {
		return 0, errors.New("모델에 필드가 없습니다")
	}

	// 기본 키 값 가져오기
	pkValue, err := getPrimaryKeyValue(model)
	if err != nil {
		return 0, err
	}

	// 업데이트 쿼리 생성
	setClause := make([]string, len(fields))
	for i, field := range fields {
		setClause[i] = fmt.Sprintf("%s = ?", field)
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?",
		model.TableName(),
		strings.Join(setClause, ", "),
		model.PrimaryKey())

	// 기본 키 값을 인자 목록에 추가
	values = append(values, pkValue)

	result, err := r.handler.Exec(query, values...)
	if err != nil {
		return 0, fmt.Errorf("모델 업데이트 실패: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("영향받은 행 수 가져오기 실패: %w", err)
	}

	return rowsAffected, nil
}

// UpdateContext는 컨텍스트와 함께 모델을 업데이트합니다
func (r *Repository) UpdateContext(ctx context.Context, model Model) (int64, error) {
	if !r.IsConnected() {
		return 0, errors.New("데이터베이스 연결이 없습니다")
	}

	fields, values, _ := extractFieldsAndValues(model)
	if len(fields) == 0 {
		return 0, errors.New("모델에 필드가 없습니다")
	}

	// 기본 키 값 가져오기
	pkValue, err := getPrimaryKeyValue(model)
	if err != nil {
		return 0, err
	}

	// 업데이트 쿼리 생성
	setClause := make([]string, len(fields))
	for i, field := range fields {
		setClause[i] = fmt.Sprintf("%s = ?", field)
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?",
		model.TableName(),
		strings.Join(setClause, ", "),
		model.PrimaryKey())

	// 기본 키 값을 인자 목록에 추가
	values = append(values, pkValue)

	result, err := r.handler.ExecContext(ctx, query, values...)
	if err != nil {
		return 0, fmt.Errorf("모델 업데이트 실패 (컨텍스트): %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("영향받은 행 수 가져오기 실패: %w", err)
	}

	return rowsAffected, nil
}

// Delete는 모델을 삭제합니다
func (r *Repository) Delete(model Model) (int64, error) {
	if !r.IsConnected() {
		return 0, errors.New("데이터베이스 연결이 없습니다")
	}

	// 기본 키 값 가져오기
	pkValue, err := getPrimaryKeyValue(model)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = ?",
		model.TableName(),
		model.PrimaryKey())

	result, err := r.handler.Exec(query, pkValue)
	if err != nil {
		return 0, fmt.Errorf("모델 삭제 실패: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("영향받은 행 수 가져오기 실패: %w", err)
	}

	return rowsAffected, nil
}

// DeleteContext는 컨텍스트와 함께 모델을 삭제합니다
func (r *Repository) DeleteContext(ctx context.Context, model Model) (int64, error) {
	if !r.IsConnected() {
		return 0, errors.New("데이터베이스 연결이 없습니다")
	}

	// 기본 키 값 가져오기
	pkValue, err := getPrimaryKeyValue(model)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = ?",
		model.TableName(),
		model.PrimaryKey())

	result, err := r.handler.ExecContext(ctx, query, pkValue)
	if err != nil {
		return 0, fmt.Errorf("모델 삭제 실패 (컨텍스트): %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("영향받은 행 수 가져오기 실패: %w", err)
	}

	return rowsAffected, nil
}

// DeleteByID는 ID로 모델을 삭제합니다
func (r *Repository) DeleteByID(model Model, id interface{}) (int64, error) {
	if !r.IsConnected() {
		return 0, errors.New("데이터베이스 연결이 없습니다")
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = ?",
		model.TableName(),
		model.PrimaryKey())

	result, err := r.handler.Exec(query, id)
	if err != nil {
		return 0, fmt.Errorf("ID로 모델 삭제 실패: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("영향받은 행 수 가져오기 실패: %w", err)
	}

	return rowsAffected, nil
}

// DeleteByIDContext는 컨텍스트와 함께 ID로 모델을 삭제합니다
func (r *Repository) DeleteByIDContext(ctx context.Context, model Model, id interface{}) (int64, error) {
	if !r.IsConnected() {
		return 0, errors.New("데이터베이스 연결이 없습니다")
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = ?",
		model.TableName(),
		model.PrimaryKey())

	result, err := r.handler.ExecContext(ctx, query, id)
	if err != nil {
		return 0, fmt.Errorf("ID로 모델 삭제 실패 (컨텍스트): %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("영향받은 행 수 가져오기 실패: %w", err)
	}

	return rowsAffected, nil
}

// Count는 모델의 총 개수를 반환합니다
func (r *Repository) Count(model Model) (int64, error) {
	if !r.IsConnected() {
		return 0, errors.New("데이터베이스 연결이 없습니다")
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", model.TableName())

	var count int64
	err := r.handler.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("모델 개수 조회 실패: %w", err)
	}

	return count, nil
}

// CountContext는 컨텍스트와 함께 모델의 총 개수를 반환합니다
func (r *Repository) CountContext(ctx context.Context, model Model) (int64, error) {
	if !r.IsConnected() {
		return 0, errors.New("데이터베이스 연결이 없습니다")
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", model.TableName())

	var count int64
	err := r.handler.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("모델 개수 조회 실패 (컨텍스트): %w", err)
	}

	return count, nil
}

// Query는 사용자 정의 쿼리를 실행합니다
func (r *Repository) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if !r.IsConnected() {
		return nil, errors.New("데이터베이스 연결이 없습니다")
	}

	return r.handler.Query(query, args...)
}

// QueryContext는 컨텍스트와 함께 사용자 정의 쿼리를 실행합니다
func (r *Repository) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if !r.IsConnected() {
		return nil, errors.New("데이터베이스 연결이 없습니다")
	}

	return r.handler.QueryContext(ctx, query, args...)
}

// Exec는 사용자 정의 쿼리를 실행합니다
func (r *Repository) Exec(query string, args ...interface{}) (sql.Result, error) {
	if !r.IsConnected() {
		return nil, errors.New("데이터베이스 연결이 없습니다")
	}

	return r.handler.Exec(query, args...)
}

// ExecContext는 컨텍스트와 함께 사용자 정의 쿼리를 실행합니다
func (r *Repository) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if !r.IsConnected() {
		return nil, errors.New("데이터베이스 연결이 없습니다")
	}

	return r.handler.ExecContext(ctx, query, args...)
}

// Transaction은 트랜잭션을 시작합니다
func (r *Repository) Transaction() (*Transaction, error) {
	if !r.IsConnected() {
		return nil, errors.New("데이터베이스 연결이 없습니다")
	}

	return r.handler.Begin()
}

// TransactionContext는 컨텍스트와 함께 트랜잭션을 시작합니다
func (r *Repository) TransactionContext(ctx context.Context, opts *sql.TxOptions) (*Transaction, error) {
	if !r.IsConnected() {
		return nil, errors.New("데이터베이스 연결이 없습니다")
	}

	return r.handler.BeginTx(ctx, opts)
}

// 유틸리티 함수들

// extractFields는 모델에서 필드 이름을 추출합니다
func extractFields(model interface{}) []string {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	fields := make([]string, 0, val.NumField())

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)

		// 비공개 필드 무시
		if field.PkgPath != "" {
			continue
		}

		// 태그에서 필드 이름 가져오기
		dbTag := field.Tag.Get("db")
		if dbTag == "-" {
			continue
		}

		if dbTag == "" {
			// 태그가 없으면 필드 이름을 소문자로 변환
			dbTag = strings.ToLower(field.Name)
		}

		fields = append(fields, dbTag)
	}

	return fields
}

// extractFieldsAndValues는 모델에서 필드 이름과 값을 추출합니다
func extractFieldsAndValues(model interface{}) ([]string, []interface{}, []string) {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, nil, nil
	}

	typ := val.Type()
	fields := make([]string, 0, val.NumField())
	values := make([]interface{}, 0, val.NumField())
	placeholders := make([]string, 0, val.NumField())

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)

		// 비공개 필드 무시
		if field.PkgPath != "" {
			continue
		}

		// 태그에서 필드 이름 가져오기
		dbTag := field.Tag.Get("db")
		if dbTag == "-" {
			continue
		}

		if dbTag == "" {
			// 태그가 없으면 필드 이름을 소문자로 변환
			dbTag = strings.ToLower(field.Name)
		}

		// 기본 키가 자동 증가인 경우 삽입 시 무시
		if field.Tag.Get("pk") == "auto" && val.Field(i).IsZero() {
			continue
		}

		fields = append(fields, dbTag)
		values = append(values, val.Field(i).Interface())
		placeholders = append(placeholders, "?")
	}

	return fields, values, placeholders
}

// scanModel은 행에서 모델로 데이터를 스캔합니다
func scanModel(scanner interface{}, model interface{}) error {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return errors.New("모델은 구조체여야 합니다")
	}

	typ := val.Type()
	fields := make([]interface{}, 0, val.NumField())

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)

		// 비공개 필드 무시
		if field.PkgPath != "" {
			continue
		}

		// 태그에서 필드 이름 가져오기
		dbTag := field.Tag.Get("db")
		if dbTag == "-" {
			continue
		}

		// 필드 포인터 추가
		fields = append(fields, val.Field(i).Addr().Interface())
	}

	var err error
	switch s := scanner.(type) {
	case *sql.Row:
		err = s.Scan(fields...)
	case *sql.Rows:
		err = s.Scan(fields...)
	default:
		return errors.New("지원되지 않는 스캐너 유형")
	}

	return err
}

// getPrimaryKeyValue는 모델에서 기본 키 값을 가져옵니다
func getPrimaryKeyValue(model Model) (interface{}, error) {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, errors.New("모델은 구조체여야 합니다")
	}

	typ := val.Type()
	pkName := model.PrimaryKey()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)

		// 태그에서 필드 이름 가져오기
		dbTag := field.Tag.Get("db")
		if dbTag == "" {
			dbTag = strings.ToLower(field.Name)
		}

		if dbTag == pkName || field.Tag.Get("pk") != "" {
			return val.Field(i).Interface(), nil
		}
	}

	return nil, fmt.Errorf("기본 키 '%s'를 찾을 수 없습니다", pkName)
}

// BaseModel은 기본 모델 구현을 제공합니다
type BaseModel struct {
	ID        int64     `db:"id" pk:"auto"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// PrimaryKey는 기본 키 이름을 반환합니다
func (m *BaseModel) PrimaryKey() string {
	return "id"
}

// BeforeCreate는 생성 전에 호출됩니다
func (m *BaseModel) BeforeCreate() {
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now
}

// BeforeUpdate는 업데이트 전에 호출됩니다
func (m *BaseModel) BeforeUpdate() {
	m.UpdatedAt = time.Now()
}
