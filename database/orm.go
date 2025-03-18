package database

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"

	"database/sql"
	"go_server_framework/loghandle"
)

/*
ORM 패키지 사용 가이드
=====================

이 ORM 패키지는 SQL 쿼리를 직접 작성하지 않고도 Go 구조체를 사용하여 데이터베이스 테이블을 조작할 수 있는 방법을 제공합니다.

기본 사용법:
1. 구조체 정의:
   ```go
   type User struct {
     ID        int64     `pk:"true" auto:"true"`  // 자동으로 id 컬럼으로 변환
     Name      string                            // 자동으로 name 컬럼으로 변환
     Email     string    `db:"email_address"`    // 명시적 컬럼명 지정
     CreatedAt time.Time `default:"CURRENT_TIMESTAMP"`
   }
   ```

2. Repository 생성:
   ```go
   repo, err := NewRepository()
   ```

3. 레코드 조회:
   ```go
   // 단일 레코드
   user := User{}
   err := repo.FindOne(&user, map[string]interface{}{"id": 1})

   // 여러 레코드
   var users []User
   err := repo.FindAll(&users, &FindOptions{
     Where: map[string]interface{}{"active": true},
     OrderBy: []string{"created_at DESC"},
     Limit: 10,
   })
   ```

4. 레코드 생성/수정/삭제:
   ```go
   // 생성
   id, err := repo.InsertStruct(user)

   // 업데이트
   affected, err := repo.UpdateStruct(user, map[string]interface{}{"id": user.ID})

   // 삭제
   affected, err := repo.DeleteStruct(user, map[string]interface{}{"id": user.ID})
   ```

5. 배치 작업:
   ```go
   batch, _ := repo.NewBatch()
   batch.AddInsert(user1)
   batch.AddUpdate(user2, map[string]interface{}{"id": user2.ID})
   batch.AddDelete(user3, map[string]interface{}{"id": 3})
   affected, err := batch.Execute()
   ```

고급 기능:
- 쿼리 필터링: `"field__op": value` 형태 지원 (예: `"age__gt": 18`)
  * gt: >
  * gte: >=
  * lt: <
  * lte: <=
  * ne: <>
  * like: LIKE
  * in: IN (value는 슬라이스)
  * null: IS NULL/IS NOT NULL (value는 bool)

- 정렬: `[]string{"field DESC", "field2"}`
- 특정 컬럼만 조회: `Columns: []string{"id", "name"}`
- DISTINCT, GROUP BY, HAVING 조건 지원

자세한 예제는 `NewORMExample()` 함수를 참조하세요.
*/

// SQL 로그 출력 설정
var EnableSQLLogging bool = false

// SQL 로그 출력 함수
func logSQL(query string, params []interface{}) {
	if EnableSQLLogging {
		loghandle.Info("SQL: %s", query)
		loghandle.Info("Parameters: %v", params)
	}
}

// Zero 값인지 확인
func isZeroValue(v reflect.Value) bool {
	// 타입이 없는 경우 (nil)
	if !v.IsValid() {
		return true
	}

	// 필드 타입에 따른 Zero 값 확인
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return v.IsNil()
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Array:
		// 배열의 모든 요소가 Zero 값인지 확인
		for i := 0; i < v.Len(); i++ {
			if !isZeroValue(v.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Struct:
		// time.Time 타입 처리
		if t, ok := v.Interface().(time.Time); ok {
			return t.IsZero()
		}
		// 구조체의 모든 필드가 Zero 값인지 확인
		for i := 0; i < v.NumField(); i++ {
			if !isZeroValue(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		// 알 수 없는 타입은 Zero 값이 아니라고 가정
		return false
	}
}

// 테이블 이름을 스네이크 케이스로 변환하는 함수
func getTableName(obj interface{}) string {
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// 기본값으로 타입 이름을 스네이크 케이스로 변환
	tableName := toSnakeCase(t.Name())

	// TableName 메서드가 있는지 확인
	tableNameMethod, ok := t.MethodByName("TableName")
	if ok && tableNameMethod.Type.NumIn() == 1 && tableNameMethod.Type.NumOut() == 1 && tableNameMethod.Type.Out(0).Kind() == reflect.String {
		// 메서드가 있으면 호출하여 결과 반환
		result := tableNameMethod.Func.Call([]reflect.Value{reflect.ValueOf(obj)})
		if len(result) > 0 && result[0].Kind() == reflect.String {
			return result[0].String()
		}
	}

	return tableName
}

// 필드를 DB 컬럼 이름으로 변환하는 함수
func getDBFieldName(field reflect.StructField) string {
	// db 태그가 있으면 해당 값 사용
	if tag, ok := field.Tag.Lookup("db"); ok {
		return tag
	}

	// 태그가 없으면 필드 이름을 스네이크 케이스로 변환
	return toSnakeCase(field.Name)
}

// 카멜 케이스를 스네이크 케이스로 변환하는 함수
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			// 첫 번째 문자이고 대문자인 경우 소문자로 변경
			if i == 0 {
				result.WriteRune(unicode.ToLower(r))
			} else {
				// 이전에 언더스코어를 추가하고 소문자로 변경
				result.WriteRune('_')
				result.WriteRune(unicode.ToLower(r))
			}
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// 기본 키 필드 찾기
func getPrimaryKeyField(t reflect.Type) (string, bool, bool) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if _, ok := field.Tag.Lookup("pk"); ok {
			auto := false
			if _, autoOk := field.Tag.Lookup("auto"); autoOk {
				auto = true
			}
			return getDBFieldName(field), true, auto
		}
	}
	return "", false, false
}

// 필드를 SQL 타입으로 변환하는 함수
func getSQLType(field reflect.StructField) string {
	switch field.Type.Kind() {
	case reflect.Int, reflect.Int32:
		return "int"
	case reflect.Int8:
		return "tinyint"
	case reflect.Int16:
		return "smallint"
	case reflect.Int64:
		return "bigint"
	case reflect.Uint, reflect.Uint32:
		return "int unsigned"
	case reflect.Uint8:
		return "tinyint unsigned"
	case reflect.Uint16:
		return "smallint unsigned"
	case reflect.Uint64:
		return "bigint unsigned"
	case reflect.Float32:
		return "float"
	case reflect.Float64:
		return "double"
	case reflect.Bool:
		return "tinyint(1)"
	case reflect.String:
		// 문자열 길이 태그가 있으면 사용
		if length, ok := field.Tag.Lookup("length"); ok {
			return "varchar(" + length + ")"
		}
		// 기본 길이 설정
		return "varchar(255)"
	case reflect.Struct:
		// time.Time 타입 처리
		if field.Type == reflect.TypeOf(time.Time{}) {
			return "datetime"
		}
	}

	// 기본값
	return "varchar(255)"
}

// 기본값 설정
func getDefaultValue(field reflect.StructField) string {
	if defaultVal, ok := field.Tag.Lookup("default"); ok {
		if defaultVal == "CURRENT_TIMESTAMP" {
			return " DEFAULT CURRENT_TIMESTAMP"
		}
		if field.Type.Kind() == reflect.String {
			return fmt.Sprintf(" DEFAULT '%s'", defaultVal)
		}
		return " DEFAULT " + defaultVal
	}

	// 기본값 없음
	return ""
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

// getFieldPointers는 구조체 필드의 포인터 슬라이스를 반환합니다
func getFieldPointers(dest interface{}) []interface{} {
	v := reflect.ValueOf(dest)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	pointers := make([]interface{}, 0, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// 비공개 필드 무시
		if field.PkgPath != "" {
			continue
		}

		pointers = append(pointers, v.Field(i).Addr().Interface())
	}

	return pointers
}

// 테이블 생성 함수 - 구조체를 기반으로 CREATE TABLE 문 생성
// ifNotExists: true인 경우 테이블이 존재하지 않을 때만 생성 (CREATE TABLE IF NOT EXISTS)
// dropIfExists: 가변 매개변수로, 첫 번째 값이 true인 경우 기존 테이블 삭제 후 생성, 두 번째 값이 true인 경우 삭제 전 백업 테이블 생성
func (r *Repository) CreateTableFromStruct(obj interface{}, ifNotExists bool, dropIfExists ...bool) error {
	if !r.IsConnected() {
		return errors.New("데이터베이스 연결이 없습니다")
	}

	t := reflect.TypeOf(obj)
	if t.Kind() != reflect.Struct {
		return errors.New("객체는 구조체여야 합니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	// dropIfExists 옵션이 활성화된 경우 테이블을 먼저 삭제
	if len(dropIfExists) > 0 && dropIfExists[0] {
		ifNotExists = false // dropIfExists가 true인 경우 ifNotExists는 의미가 없음

		// 백업 옵션 확인 (2번째 옵션이 있고 true면 백업 생성)
		if len(dropIfExists) > 1 && dropIfExists[1] {
			backupTableName := tableName + "_backup_" + time.Now().Format("20060102_150405")

			// 백업 테이블 생성
			backupQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` LIKE `%s`", backupTableName, tableName)
			if EnableSQLLogging {
				params := []interface{}{}
				logSQL(backupQuery, params)
			}

			_, err := r.handler.Exec(backupQuery)
			if err != nil {
				return fmt.Errorf("백업 테이블 생성 실패: %w", err)
			}

			// 데이터 복사
			copyDataQuery := fmt.Sprintf("INSERT INTO `%s` SELECT * FROM `%s`", backupTableName, tableName)
			if EnableSQLLogging {
				params := []interface{}{}
				logSQL(copyDataQuery, params)
			}

			_, err = r.handler.Exec(copyDataQuery)
			if err != nil {
				return fmt.Errorf("데이터 복사 실패: %w", err)
			}

			loghandle.Info("테이블 '%s'의 백업 '%s' 생성 성공", tableName, backupTableName)
		}

		dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName)

		// 로깅
		if EnableSQLLogging {
			params := []interface{}{}
			logSQL(dropQuery, params)
		}

		// 테이블 삭제 쿼리 실행
		_, err := r.handler.Exec(dropQuery)
		if err != nil {
			return fmt.Errorf("테이블 삭제 실패: %w", err)
		}

		loghandle.Info("테이블 '%s' 삭제 성공", tableName)
	}

	var columns []string
	var primaryKey string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		dbFieldName := getDBFieldName(field)
		sqlType := getSQLType(field)
		nullable := " NOT NULL"
		defaultValue := getDefaultValue(field)

		if _, ok := field.Tag.Lookup("Null"); ok {
			nullable = ""
		}

		columnDef := fmt.Sprintf("`%s` %s%s%s", dbFieldName, sqlType, nullable, defaultValue)
		columns = append(columns, columnDef)

		if _, ok := field.Tag.Lookup("pk"); ok {
			primaryKey = fmt.Sprintf("PRIMARY KEY (`%s`)", dbFieldName)
			if _, autoOk := field.Tag.Lookup("auto"); autoOk {
				columns[i] = fmt.Sprintf("`%s` %s%s%s AUTO_INCREMENT", dbFieldName, sqlType, nullable, defaultValue)
			}
		}
	}

	// 기본 키가 있으면 추가
	if primaryKey != "" {
		columns = append(columns, primaryKey)
	}

	existsClause := ""
	if ifNotExists {
		existsClause = "IF NOT EXISTS "
	}

	query := fmt.Sprintf("CREATE TABLE %s`%s` (\n  %s\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci",
		existsClause, tableName, strings.Join(columns, ",\n  "))

	// 로깅
	if EnableSQLLogging {
		params := []interface{}{}
		logSQL(query, params)
	}

	// 쿼리 실행
	_, err := r.handler.Exec(query)
	return err
}

// FindOneByQuery는 사용자 정의 쿼리로 단일 레코드를 조회합니다
// 경고: 직접 SQL을 작성하는 대신 Find 또는 FindOne 함수 사용을 권장합니다.
// 예: repo.FindOne(&user, map[string]interface{}{"id": 1})
func (r *Repository) FindOneByQuery(dest interface{}, query string, args ...interface{}) error {
	if !r.IsConnected() {
		return errors.New("데이터베이스 연결이 없습니다")
	}

	if EnableSQLLogging {
		logSQL(query, args)
	}

	// 쿼리 실행
	row := r.handler.QueryRow(query, args...)

	// 결과를 구조체로 스캔
	if err := row.Scan(getFieldPointers(dest)...); err != nil {
		return err
	}

	return nil
}

// FindAllByQuery는 사용자 정의 쿼리로 여러 레코드를 조회합니다
// 경고: 직접 SQL을 작성하는 대신 Find 또는 FindAll 함수 사용을 권장합니다.
// 예: repo.FindAll(&users, &FindOptions{Where: map[string]interface{}{"active": true}})
func (r *Repository) FindAllByQuery(dest interface{}, query string, args ...interface{}) error {
	if !r.IsConnected() {
		return errors.New("데이터베이스 연결이 없습니다")
	}

	// dest는 슬라이스 포인터여야 함
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return errors.New("dest는 슬라이스 포인터여야 합니다")
	}

	if EnableSQLLogging {
		logSQL(query, args)
	}

	// 쿼리 실행
	rows, err := r.handler.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	// 슬라이스 요소 타입 가져오기
	sliceElemType := destValue.Elem().Type().Elem()

	// 결과를 슬라이스에 추가
	sliceValue := destValue.Elem()
	for rows.Next() {
		// 새 요소 생성
		var newElem reflect.Value
		if sliceElemType.Kind() == reflect.Ptr {
			newElem = reflect.New(sliceElemType.Elem())
		} else {
			newElem = reflect.New(sliceElemType).Elem()
		}

		// 스캔
		err := rows.Scan(getFieldPointers(newElem.Interface())...)
		if err != nil {
			return err
		}

		// 슬라이스에 추가
		if sliceElemType.Kind() == reflect.Ptr {
			sliceValue = reflect.Append(sliceValue, newElem)
		} else {
			sliceValue = reflect.Append(sliceValue, newElem)
		}
	}

	destValue.Elem().Set(sliceValue)
	return nil
}

// InsertStruct는 구조체를 데이터베이스에 삽입합니다
func (r *Repository) InsertStruct(obj interface{}) (int64, error) {
	if !r.IsConnected() {
		return 0, errors.New("데이터베이스 연결이 없습니다")
	}

	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return 0, errors.New("객체는 구조체여야 합니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	// 필드와 값 추출
	var columns []string
	var placeholders []string
	var params []interface{}

	pkField, _, isAuto := getPrimaryKeyField(t)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// 비공개 필드 무시
		if field.PkgPath != "" {
			continue
		}

		dbFieldName := getDBFieldName(field)

		// 자동 증가 PK는 INSERT 부분에서 제외
		if isAuto && dbFieldName == pkField && isZeroValue(fieldValue) {
			continue
		}

		columns = append(columns, "`"+dbFieldName+"`")
		placeholders = append(placeholders, "?")
		params = append(params, fieldValue.Interface())
	}

	// 쿼리 생성
	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	if EnableSQLLogging {
		logSQL(query, params)
	}

	// 쿼리 실행
	result, err := r.handler.Exec(query, params...)
	if err != nil {
		return 0, err
	}

	// 마지막 삽입 ID 반환
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

// UpdateStruct는 구조체를 데이터베이스에서 업데이트합니다
func (r *Repository) UpdateStruct(obj interface{}, where map[string]interface{}) (int64, error) {
	if !r.IsConnected() {
		return 0, errors.New("데이터베이스 연결이 없습니다")
	}

	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return 0, errors.New("객체는 구조체여야 합니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	// 필드와 값 추출
	var setFields []string
	var params []interface{}

	pkField, _, _ := getPrimaryKeyField(t)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// 비공개 필드 무시
		if field.PkgPath != "" {
			continue
		}

		dbFieldName := getDBFieldName(field)

		// PK는 업데이트하지 않음
		if dbFieldName == pkField {
			continue
		}

		// Zero 값이 아닌 필드만 업데이트
		if !isZeroValue(fieldValue) {
			setFields = append(setFields, fmt.Sprintf("`%s` = ?", dbFieldName))
			params = append(params, fieldValue.Interface())
		}
	}

	if len(setFields) == 0 {
		return 0, errors.New("업데이트할 필드가 없습니다")
	}

	// 기본 쿼리 생성
	query := fmt.Sprintf("UPDATE `%s` SET %s", tableName, strings.Join(setFields, ", "))

	// WHERE 절 추가
	if len(where) > 0 {
		conditions := []string{}
		for field, value := range where {
			conditions = append(conditions, fmt.Sprintf("`%s` = ?", field))
			params = append(params, value)
		}
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	if EnableSQLLogging {
		logSQL(query, params)
	}

	// 쿼리 실행
	result, err := r.handler.Exec(query, params...)
	if err != nil {
		return 0, err
	}

	// 영향받은 행 수 반환
	return result.RowsAffected()
}

// UpsertStruct는 구조체를 삽입하거나 중복 키 충돌 시 업데이트합니다
func (r *Repository) UpsertStruct(obj interface{}) (int64, error) {
	if !r.IsConnected() {
		return 0, errors.New("데이터베이스 연결이 없습니다")
	}

	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return 0, errors.New("객체는 구조체여야 합니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	pkField, hasPK, isAuto := getPrimaryKeyField(t)
	if !hasPK {
		return 0, errors.New("UPSERT에는 PK 필드가 필요합니다")
	}

	// 모든 필드와 값을 수집
	var columns []string
	var placeholders []string
	var params []interface{}
	var updateClauses []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// 비공개 필드 무시
		if field.PkgPath != "" {
			continue
		}

		dbFieldName := getDBFieldName(field)

		// 자동 증가 PK는 INSERT 부분에서 제외
		if isAuto && dbFieldName == pkField && isZeroValue(fieldValue) {
			continue
		}

		columns = append(columns, "`"+dbFieldName+"`")
		placeholders = append(placeholders, "?")
		params = append(params, fieldValue.Interface())

		// PK는 업데이트 부분에서 제외
		if dbFieldName != pkField {
			updateClauses = append(updateClauses, fmt.Sprintf("`%s`=VALUES(`%s`)", dbFieldName, dbFieldName))
		}
	}

	// UPSERT 쿼리 구성 (MySQL 방식)
	query := fmt.Sprintf(
		"INSERT INTO `%s` (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(updateClauses, ", "),
	)

	if EnableSQLLogging {
		logSQL(query, params)
	}

	// 쿼리 실행
	result, err := r.handler.Exec(query, params...)
	if err != nil {
		return 0, err
	}

	// 영향받은, 또는 마지막으로 삽입된 ID 반환
	lastID, err := result.LastInsertId()
	if err != nil {
		// LastInsertId 가져오기 실패 시 영향받은 행 수 반환
		affected, affErr := result.RowsAffected()
		if affErr != nil {
			return 0, affErr
		}
		return affected, nil
	}

	return lastID, nil
}

// DeleteStruct는 구조체를 데이터베이스에서 삭제합니다
func (r *Repository) DeleteStruct(obj interface{}, where map[string]interface{}) (int64, error) {
	if !r.IsConnected() {
		return 0, errors.New("데이터베이스 연결이 없습니다")
	}

	t := reflect.TypeOf(obj)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return 0, errors.New("객체는 구조체여야 합니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	var params []interface{}

	// 기본 쿼리 생성
	query := fmt.Sprintf("DELETE FROM `%s`", tableName)

	// WHERE 절 추가
	if len(where) > 0 {
		conditions := []string{}
		for field, value := range where {
			conditions = append(conditions, fmt.Sprintf("`%s` = ?", field))
			params = append(params, value)
		}
		query += " WHERE " + strings.Join(conditions, " AND ")
	} else {
		return 0, errors.New("삭제 쿼리에는 WHERE 조건이 필요합니다")
	}

	if EnableSQLLogging {
		logSQL(query, params)
	}

	// 쿼리 실행
	result, err := r.handler.Exec(query, params...)
	if err != nil {
		return 0, err
	}

	// 영향받은 행 수 반환
	return result.RowsAffected()
}

// BatchJobType은 배치 작업의 유형을 정의합니다
type BatchJobType int

const (
	BatchInsert BatchJobType = iota
	BatchUpdate
	BatchDelete
	BatchUpsert
)

// BatchJob은 배치 작업의 정보를 담고 있습니다
type BatchJob struct {
	JobType BatchJobType
	Query   string
	Params  []interface{}
}

// 배치 작업 처리를 위한 구조체
type Batch struct {
	repo *Repository
	tx   *Transaction
	jobs []BatchJob
}

// NewBatch는 새 배치 작업을 생성합니다
func (r *Repository) NewBatch() (*Batch, error) {
	if !r.IsConnected() {
		return nil, errors.New("데이터베이스 연결이 없습니다")
	}

	tx, err := r.handler.Begin()
	if err != nil {
		return nil, err
	}

	return &Batch{
		repo: r,
		tx:   tx,
		jobs: make([]BatchJob, 0),
	}, nil
}

// AddInsert는 배치에 INSERT 작업을 추가합니다
func (b *Batch) AddInsert(obj interface{}) error {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return errors.New("객체는 구조체여야 합니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	pkField, _, isAuto := getPrimaryKeyField(t)

	// 모든 필드와 값 수집
	var columns []string
	var placeholders []string
	var params []interface{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// 비공개 필드 무시
		if field.PkgPath != "" {
			continue
		}

		dbFieldName := getDBFieldName(field)

		// 자동 증가 PK는 건너뜀
		if isAuto && dbFieldName == pkField && isZeroValue(fieldValue) {
			continue
		}

		columns = append(columns, "`"+dbFieldName+"`")
		placeholders = append(placeholders, "?")
		params = append(params, fieldValue.Interface())
	}

	// 쿼리 구성
	query := fmt.Sprintf(
		"INSERT INTO `%s` (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	// 배치에 작업 추가
	b.jobs = append(b.jobs, BatchJob{
		JobType: BatchInsert,
		Query:   query,
		Params:  params,
	})

	return nil
}

// AddUpdate는 배치에 UPDATE 작업을 추가합니다
func (b *Batch) AddUpdate(obj interface{}, where map[string]interface{}) error {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return errors.New("객체는 구조체여야 합니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	pkField, hasPK, _ := getPrimaryKeyField(t)

	// where 맵이 비어 있고 PK가 있으면 PK를 사용
	if len(where) == 0 && hasPK {
		// PK 값 찾기
		var pkValue interface{}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if getDBFieldName(field) == pkField {
				pkValue = v.Field(i).Interface()
				break
			}
		}

		if !isZeroValue(reflect.ValueOf(pkValue)) {
			where = map[string]interface{}{
				pkField: pkValue,
			}
		} else {
			return errors.New("업데이트에 WHERE 조건이 지정되지 않았고 PK 값이 0입니다")
		}
	}

	if len(where) == 0 {
		return errors.New("업데이트에는 WHERE 조건이 필요합니다")
	}

	// 설정할 필드와 값 수집
	var setClauses []string
	var params []interface{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// 비공개 필드 무시
		if field.PkgPath != "" {
			continue
		}

		dbFieldName := getDBFieldName(field)

		// PK 필드는 설정하지 않음
		if dbFieldName == pkField {
			continue
		}

		setClauses = append(setClauses, fmt.Sprintf("`%s` = ?", dbFieldName))
		params = append(params, fieldValue.Interface())
	}

	// WHERE 절 파라미터 추가
	whereConditions := []string{}
	for field, value := range where {
		whereConditions = append(whereConditions, fmt.Sprintf("`%s` = ?", field))
		params = append(params, value)
	}

	// 쿼리 구성
	query := fmt.Sprintf(
		"UPDATE `%s` SET %s WHERE %s",
		tableName,
		strings.Join(setClauses, ", "),
		strings.Join(whereConditions, " AND "),
	)

	// 배치에 작업 추가
	b.jobs = append(b.jobs, BatchJob{
		JobType: BatchUpdate,
		Query:   query,
		Params:  params,
	})

	return nil
}

// AddDelete는 배치에 DELETE 작업을 추가합니다
func (b *Batch) AddDelete(obj interface{}, where map[string]interface{}) error {
	t := reflect.TypeOf(obj)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return errors.New("객체는 구조체여야 합니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	var params []interface{}

	// 기본 쿼리 생성
	query := fmt.Sprintf("DELETE FROM `%s`", tableName)

	// WHERE 절 추가
	if len(where) > 0 {
		conditions := []string{}
		for field, value := range where {
			conditions = append(conditions, fmt.Sprintf("`%s` = ?", field))
			params = append(params, value)
		}
		query += " WHERE " + strings.Join(conditions, " AND ")
	} else {
		return errors.New("삭제 쿼리에는 WHERE 조건이 필요합니다")
	}

	// 배치에 작업 추가
	b.jobs = append(b.jobs, BatchJob{
		JobType: BatchDelete,
		Query:   query,
		Params:  params,
	})

	return nil
}

// AddUpsert는 배치에 UPSERT 작업을 추가합니다
func (b *Batch) AddUpsert(obj interface{}) error {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return errors.New("객체는 구조체여야 합니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	pkField, hasPK, isAuto := getPrimaryKeyField(t)
	if !hasPK {
		return errors.New("UPSERT에는 PK 필드가 필요합니다")
	}

	// 모든 필드와 값을 수집
	var columns []string
	var placeholders []string
	var params []interface{}
	var updateClauses []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// 비공개 필드 무시
		if field.PkgPath != "" {
			continue
		}

		dbFieldName := getDBFieldName(field)

		// 자동 증가 PK는 INSERT 부분에서 제외
		if isAuto && dbFieldName == pkField && isZeroValue(fieldValue) {
			continue
		}

		columns = append(columns, "`"+dbFieldName+"`")
		placeholders = append(placeholders, "?")
		params = append(params, fieldValue.Interface())

		// PK는 업데이트 부분에서 제외
		if dbFieldName != pkField {
			updateClauses = append(updateClauses, fmt.Sprintf("`%s`=VALUES(`%s`)", dbFieldName, dbFieldName))
		}
	}

	// UPSERT 쿼리 구성 (MySQL 방식)
	query := fmt.Sprintf(
		"INSERT INTO `%s` (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(updateClauses, ", "),
	)

	// 배치에 작업 추가
	b.jobs = append(b.jobs, BatchJob{
		JobType: BatchUpsert,
		Query:   query,
		Params:  params,
	})

	return nil
}

// Execute는 배치 작업을 실행합니다
func (b *Batch) Execute() (int64, error) {
	var totalAffected int64 = 0

	for _, job := range b.jobs {
		// 로깅
		if EnableSQLLogging {
			logSQL(job.Query, job.Params)
		}

		// 작업 실행
		result, err := b.tx.Exec(job.Query, job.Params...)
		if err != nil {
			b.tx.Rollback()
			return 0, err
		}

		// 영향받은 행 수 합산
		affected, err := result.RowsAffected()
		if err != nil {
			// 영향받은 행 수를 가져올 수 없는 경우 무시
			continue
		}

		totalAffected += affected
	}

	// 트랜잭션 커밋
	err := b.tx.Commit()
	if err != nil {
		b.tx.Rollback()
		return 0, err
	}

	return totalAffected, nil
}

// Rollback은 배치 작업을 롤백합니다
func (b *Batch) Rollback() error {
	return b.tx.Rollback()
}

// IsTableExists는 테이블이 존재하는지 확인합니다
func (r *Repository) IsTableExists(tableName string) (bool, error) {
	if !r.IsConnected() {
		return false, errors.New("데이터베이스 연결이 없습니다")
	}

	// 데이터베이스 이름 가져오기
	var dbName string
	query := "SELECT DATABASE()"
	row := r.handler.QueryRow(query)
	err := row.Scan(&dbName)
	if err != nil {
		return false, err
	}

	// 테이블 존재 여부 확인
	query = `
		SELECT COUNT(*) 
		FROM information_schema.tables 
		WHERE table_schema = ? 
		AND table_name = ?
	`

	if EnableSQLLogging {
		params := []interface{}{dbName, tableName}
		logSQL(query, params)
	}

	var count int
	row = r.handler.QueryRow(query, dbName, tableName)
	err = row.Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// FindOptions는 Find 함수에 사용되는 옵션을 정의합니다
type FindOptions struct {
	Where      map[string]interface{} // WHERE 조건
	OrderBy    []string               // 정렬 필드 (필드명 또는 "필드명 DESC")
	Limit      int                    // 결과 제한
	Offset     int                    // 결과 오프셋
	Columns    []string               // 조회할 컬럼 목록 (비어있으면 모든 컬럼)
	Distinct   bool                   // DISTINCT 사용 여부
	GroupBy    []string               // GROUP BY 절
	HavingCond map[string]interface{} // HAVING 조건
}

// Result는 쿼리 결과를 저장하는 제네릭 타입입니다
type Result[T any] struct {
	Items []T
	Count int64
}

// Find는 타입 T에 대한 레코드를 조회합니다
func (r *Repository) Find(dest interface{}, options *FindOptions) error {
	if !r.IsConnected() {
		return errors.New("데이터베이스 연결이 없습니다")
	}

	t := reflect.TypeOf(dest)
	// dest는 슬라이스 포인터이거나 구조체 포인터여야 함
	if t.Kind() != reflect.Ptr {
		return errors.New("dest는 슬라이스 포인터 또는 구조체 포인터여야 합니다")
	}

	elem := t.Elem()
	isSingle := elem.Kind() != reflect.Slice // 단일 구조체인지 확인

	var structType reflect.Type
	if isSingle {
		// 단일 구조체 조회
		structType = elem
	} else {
		// 슬라이스의 요소 타입 가져오기
		structType = elem.Elem()
		if structType.Kind() == reflect.Ptr {
			structType = structType.Elem()
		}
	}

	if structType.Kind() != reflect.Struct {
		return errors.New("dest의 요소는 구조체여야 합니다")
	}

	// 테이블 이름 가져오기
	// dummy 객체를 생성하여 테이블 이름 가져오기
	dummyObj := reflect.New(structType).Interface()
	tableName := getTableName(dummyObj)

	// 컬럼 목록 구성
	var columns []string
	if options != nil && len(options.Columns) > 0 {
		columns = options.Columns
	} else {
		// 모든 필드를 컬럼으로 사용
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			// 비공개 필드 무시
			if field.PkgPath != "" {
				continue
			}
			columns = append(columns, "`"+getDBFieldName(field)+"`")
		}
	}

	// 쿼리 조합
	var queryBuilder strings.Builder
	var params []interface{}

	// SELECT 절 구성
	if options != nil && options.Distinct {
		queryBuilder.WriteString("SELECT DISTINCT ")
	} else {
		queryBuilder.WriteString("SELECT ")
	}
	queryBuilder.WriteString(strings.Join(columns, ", "))
	queryBuilder.WriteString(" FROM `")
	queryBuilder.WriteString(tableName)
	queryBuilder.WriteString("`")

	// WHERE 절 구성
	if options != nil && len(options.Where) > 0 {
		queryBuilder.WriteString(" WHERE ")

		conditions := []string{}
		for field, value := range options.Where {
			// 다양한 조건 처리 (field__op 형식 지원)
			parts := strings.Split(field, "__")
			fieldName := parts[0]
			operator := "="

			if len(parts) > 1 {
				switch parts[1] {
				case "gt":
					operator = ">"
				case "gte":
					operator = ">="
				case "lt":
					operator = "<"
				case "lte":
					operator = "<="
				case "ne":
					operator = "<>"
				case "like":
					operator = "LIKE"
				case "in":
					// IN 연산자 처리
					if reflect.TypeOf(value).Kind() == reflect.Slice {
						s := reflect.ValueOf(value)
						placeholders := make([]string, s.Len())
						for i := 0; i < s.Len(); i++ {
							placeholders[i] = "?"
							params = append(params, s.Index(i).Interface())
						}
						conditions = append(conditions, fmt.Sprintf("`%s` IN (%s)", fieldName, strings.Join(placeholders, ", ")))
						continue
					}
				case "null":
					// IS NULL 또는 IS NOT NULL 처리
					if value.(bool) {
						conditions = append(conditions, fmt.Sprintf("`%s` IS NULL", fieldName))
					} else {
						conditions = append(conditions, fmt.Sprintf("`%s` IS NOT NULL", fieldName))
					}
					continue
				}
			}

			// 기본 조건 추가
			if operator != "IN" {
				conditions = append(conditions, fmt.Sprintf("`%s` %s ?", fieldName, operator))
				params = append(params, value)
			}
		}

		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}

	// GROUP BY 절 구성
	if options != nil && len(options.GroupBy) > 0 {
		queryBuilder.WriteString(" GROUP BY ")
		groupByFields := make([]string, len(options.GroupBy))
		for i, field := range options.GroupBy {
			groupByFields[i] = "`" + field + "`"
		}
		queryBuilder.WriteString(strings.Join(groupByFields, ", "))

		// HAVING 절 구성
		if len(options.HavingCond) > 0 {
			queryBuilder.WriteString(" HAVING ")
			conditions := []string{}
			for field, value := range options.HavingCond {
				conditions = append(conditions, fmt.Sprintf("`%s` = ?", field))
				params = append(params, value)
			}
			queryBuilder.WriteString(strings.Join(conditions, " AND "))
		}
	}

	// ORDER BY 절 구성
	if options != nil && len(options.OrderBy) > 0 {
		queryBuilder.WriteString(" ORDER BY ")
		orderFields := make([]string, len(options.OrderBy))
		for i, field := range options.OrderBy {
			// 필드에 DESC 또는 ASC가 포함되어 있는지 확인
			if strings.Contains(strings.ToUpper(field), " DESC") || strings.Contains(strings.ToUpper(field), " ASC") {
				parts := strings.Fields(field)
				orderFields[i] = "`" + parts[0] + "` " + parts[1]
			} else {
				orderFields[i] = "`" + field + "`"
			}
		}
		queryBuilder.WriteString(strings.Join(orderFields, ", "))
	}

	// LIMIT 및 OFFSET 절 구성
	if options != nil && options.Limit > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" LIMIT %d", options.Limit))
		if options.Offset > 0 {
			queryBuilder.WriteString(fmt.Sprintf(" OFFSET %d", options.Offset))
		}
	}

	query := queryBuilder.String()

	if EnableSQLLogging {
		logSQL(query, params)
	}

	// 단일 레코드 조회인 경우
	if isSingle {
		// 쿼리 실행
		row := r.handler.QueryRow(query, params...)

		// 결과를 구조체로 스캔
		pointers := getFieldPointers(dest)

		// 선택된 컬럼에 대한 포인터만 추출
		if options != nil && len(options.Columns) > 0 {
			selectedPointers := make([]interface{}, 0, len(options.Columns))
			fieldMap := make(map[string]int)

			// 필드 이름 -> 인덱스 매핑 생성
			for i := 0; i < structType.NumField(); i++ {
				field := structType.Field(i)
				if field.PkgPath != "" { // 비공개 필드 무시
					continue
				}
				fieldMap[getDBFieldName(field)] = i
			}

			// 선택된 컬럼에 해당하는 포인터만 선택
			for _, col := range options.Columns {
				colName := strings.Trim(col, "`")
				if idx, ok := fieldMap[colName]; ok && idx < len(pointers) {
					selectedPointers = append(selectedPointers, pointers[idx])
				}
			}

			pointers = selectedPointers
		}

		if err := row.Scan(pointers...); err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("레코드를 찾을 수 없습니다: %w", err)
			}
			return err
		}

		return nil
	}

	// 여러 레코드 조회
	rows, err := r.handler.Query(query, params...)
	if err != nil {
		return err
	}
	defer rows.Close()

	// 슬라이스 요소 타입 가져오기
	sliceElemType := elem.Elem() // 슬라이스의 요소 타입

	// 결과를 슬라이스에 추가
	sliceValue := reflect.ValueOf(dest).Elem()

	// 컬럼 인덱스 매핑 (선택적 컬럼 처리용)
	fieldMap := make(map[string]int)
	if options != nil && len(options.Columns) > 0 {
		// 필드 이름 -> 인덱스 매핑 생성
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			if field.PkgPath != "" { // 비공개 필드 무시
				continue
			}
			fieldMap[getDBFieldName(field)] = i
		}
	}

	for rows.Next() {
		// 새 요소 생성
		var newElem reflect.Value
		if sliceElemType.Kind() == reflect.Ptr {
			newElem = reflect.New(sliceElemType.Elem())
		} else {
			newElem = reflect.New(sliceElemType).Elem()
		}

		// 스캔용 포인터 생성
		pointers := getFieldPointers(newElem.Interface())

		// 선택된 컬럼에 대한 포인터만 추출
		if options != nil && len(options.Columns) > 0 {
			selectedPointers := make([]interface{}, 0, len(options.Columns))

			// 선택된 컬럼에 해당하는 포인터만 선택
			for _, col := range options.Columns {
				colName := strings.Trim(col, "`")
				if idx, ok := fieldMap[colName]; ok && idx < len(pointers) {
					selectedPointers = append(selectedPointers, pointers[idx])
				}
			}

			pointers = selectedPointers
		}

		// 스캔
		err := rows.Scan(pointers...)
		if err != nil {
			return err
		}

		// 슬라이스에 추가
		if sliceElemType.Kind() == reflect.Ptr {
			sliceValue = reflect.Append(sliceValue, newElem)
		} else {
			sliceValue = reflect.Append(sliceValue, newElem)
		}
	}

	reflect.ValueOf(dest).Elem().Set(sliceValue)
	return nil
}

// FindOne은 단일 레코드를 조회합니다
func (r *Repository) FindOne(dest interface{}, where map[string]interface{}) error {
	options := &FindOptions{
		Where: where,
		Limit: 1,
	}
	return r.Find(dest, options)
}

// FindAll은 여러 레코드를 조회합니다
func (r *Repository) FindAll(dest interface{}, options *FindOptions) error {
	return r.Find(dest, options)
}

// Count는 조건에 맞는 레코드 수를 반환합니다
func (r *Repository) Count(obj interface{}, where map[string]interface{}) (int64, error) {
	if !r.IsConnected() {
		return 0, errors.New("데이터베이스 연결이 없습니다")
	}

	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return 0, errors.New("객체는 구조체여야 합니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	// 쿼리 조합
	query := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", tableName)
	var params []interface{}

	// WHERE 절 추가
	if len(where) > 0 {
		query += " WHERE "
		conditions := []string{}
		for field, value := range where {
			conditions = append(conditions, fmt.Sprintf("`%s` = ?", field))
			params = append(params, value)
		}
		query += strings.Join(conditions, " AND ")
	}

	if EnableSQLLogging {
		logSQL(query, params)
	}

	// 쿼리 실행
	row := r.handler.QueryRow(query, params...)

	// 결과 스캔
	var count int64
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
