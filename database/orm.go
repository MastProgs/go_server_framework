package database

import (
	"database/sql"
	"errors"
	"fmt"
	"go_server_framework/loghandle"
	"reflect"
	"strings"
	"time"
	"unicode"
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
     UpdatedAt time.Time `default:"CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"` // MySQL 자동 업데이트
     DeletedAt time.Time `Null:"true"`           // NULL 허용 필드 - 를 만들려고 했으나, 동작 안해서 항상 NOT NULL 로 테이블 관리할 것
   }
   ```

2. Repository 생성:
   ```go
   repo, err := NewRepository()
   ```

3. 레코드 조회:
   ```go
   // 단일 레코드 조회 (조회 개수와 함께 반환)
   user := User{}
   count, err := repo.FindOne(&user, map[string]interface{}{"id": 1})
   if err != nil {
       log.Printf("에러: %v", err)
   } else if count == 0 {
       log.Printf("사용자를 찾을 수 없습니다")
   } else {
       log.Printf("사용자 조회 성공: %+v", user)
   }

   // 여러 레코드 조회 (조회 개수와 함께 반환)
   var users []User
   count, err := repo.FindAll(&users, &FindOptions{
     Where: map[string]interface{}{"active": true},
     OrderBy: []string{"created_at DESC"},
     Limit: 10,
   })
   if err != nil {
       log.Printf("에러: %v", err)
   } else {
       log.Printf("조회된 사용자 수: %d", count)
   }

   // 사용자 정의 쿼리 (조회 개수와 함께 반환)
   count, err := repo.FindOneByQuery(&user, "SELECT * FROM users WHERE email = ?", "test@example.com")
   count, err := repo.FindAllByQuery(&users, "SELECT * FROM users WHERE created_at > ?", time.Now().AddDate(0, -1, 0))
   ```

4. 레코드 생성/수정/삭제:
   ```go
   // 생성
   id, err := repo.InsertStruct(user)

   // 업데이트 (전체 필드)
   affected, err := repo.UpdateStruct(user, map[string]interface{}{"id": user.ID})

   // 선택적 필드 업데이트 (맵 기반)
   fieldsToUpdate := map[string]interface{}{
     "name": "새 이름",
     "email": "new@example.com",
   }
   affected, err := repo.UpdateFields("users", fieldsToUpdate, map[string]interface{}{"id": user.ID})

   // 선택적 필드 업데이트 (필드 이름 지정)
   affected, err := repo.UpdateFieldsByStruct(user, []string{"Name", "Email"}, map[string]interface{}{"id": user.ID})

   // Zero가 아닌 필드만 업데이트
   modifiedUser := User{Name: "새 이름"} // Value, Active 등 zero value인 필드는 업데이트되지 않음
   affected, err := repo.UpdateNonZero(modifiedUser, map[string]interface{}{"id": user.ID})

   // 원본과 비교하여 변경된 필드만 업데이트
   affected, err := repo.UpdateNonZero(modifiedUser, map[string]interface{}{"id": user.ID}, originalUser)

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
  * ilike: LIKE (대소문자 무시, MySQL에서는 LOWER 함수 사용)
  * in: IN (value는 슬라이스)
  * notin: NOT IN (value는 슬라이스)
  * null: IS NULL/IS NOT NULL (value는 bool)
  * between: BETWEEN (value는 [시작값, 끝값] 슬라이스)
  * contains: 문자열 포함 검사 (LIKE %값%)
  * startswith: 문자열 시작 검사 (LIKE 값%)
  * endswith: 문자열 끝 검사 (LIKE %값)
  * regex: 정규식 검사 (MySQL REGEXP)

- 정렬: `[]string{"field DESC", "field2"}`
- 특정 컬럼만 조회: `Columns: []string{"id", "name"}`
- DISTINCT, GROUP BY, HAVING 조건 지원
- UpdatedAt 자동 갱신:
  * `default:"CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"` 태그로 MySQL의 자동 업데이트 활용
  * 테이블 생성 시 UpdatedAt 필드를 자동으로 감지하여 해당 설정 적용

필드 태그 옵션:
- `pk:"true"` - 기본 키 필드
- `auto:"true"` - 자동 증가(auto increment) 필드
- `db:"column_name"` - 데이터베이스 컬럼 이름 지정
- `default:"CURRENT_TIMESTAMP"` - 기본값 설정
- `default:"CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"` - 생성 및 업데이트 시 자동 갱신
- `Null:"true"` - NULL 값 허용
- `auto:"true"` - UpdatedAt 필드에 사용하면 자동 업데이트에서 제외 (updateNonZero 메서드 사용 시)
- `length:"255"` - VARCHAR 필드의 길이 지정

참고:
- MySQL의 SQL 모드에 따라 datetime 필드의 zero value 처리가 달라질 수 있습니다.
- 기본적으로 프레임워크는 연결 시 NO_ZERO_DATE 및 NO_ZERO_IN_DATE 옵션을 비활성화하여
  zero datetime 값이 처리될 수 있게 합니다.
- UpdateNonZero 메서드를 사용하면 zero value가 아닌 필드만 업데이트할 수 있습니다. 이는 부분 업데이트를
  더 직관적으로 구현할 수 있게 해줍니다.
- 원본 객체를 제공하면 변경된 필드만 업데이트할 수도 있습니다.

자세한 예제는 `NewORMExample()` 함수를 참조하세요.
*/

// SQL 로그 출력 설정
var EnableSQLLogging bool = false

// SQL 로그 출력 함수
func logSQL(query string, params []interface{}) {
	if !EnableSQLLogging {
		// 로깅이 비활성화되어 있어도 중요한 SQL 문은 항상 로그로 남김
		if strings.Contains(strings.ToUpper(query), "INSERT") ||
			strings.Contains(strings.ToUpper(query), "UPDATE") ||
			strings.Contains(strings.ToUpper(query), "DELETE") {
			loghandle.Info("SQL: %s", query)
			loghandle.Info("Parameters: %v", params)
		}
		return
	}

	// 로깅이 활성화된 경우 모든 쿼리를 Debug 레벨로 출력
	loghandle.Debug("SQL: %s", query)
	loghandle.Debug("Parameters: %v", params)

	// 중요한 SQL 작업은 Info 레벨로도 출력
	if strings.Contains(strings.ToUpper(query), "INSERT") ||
		strings.Contains(strings.ToUpper(query), "UPDATE") ||
		strings.Contains(strings.ToUpper(query), "DELETE") {
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

func SetZero(obj interface{}) {
	v := reflect.ValueOf(obj)

	// 포인터가 아니면 패닉 방지
	if v.Kind() != reflect.Ptr {
		return
	}

	// nil 포인터 확인
	if v.IsNil() {
		return
	}

	// 실제 값으로 접근
	v = v.Elem()

	// 구조체가 아니면 리턴
	if v.Kind() != reflect.Struct {
		return
	}

	// 모든 필드를 순회하면서 제로값으로 설정
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		// 필드가 설정 가능한지 확인
		if !field.CanSet() {
			continue
		}

		// 필드 타입에 따라 제로값 설정
		switch field.Kind() {
		case reflect.String:
			field.SetString("")
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			field.SetInt(0)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			field.SetUint(0)
		case reflect.Float32, reflect.Float64:
			field.SetFloat(0)
		case reflect.Bool:
			field.SetBool(false)
		case reflect.Slice, reflect.Map, reflect.Chan:
			field.Set(reflect.Zero(field.Type()))
		case reflect.Interface, reflect.Ptr:
			field.Set(reflect.Zero(field.Type()))
		case reflect.Array:
			// 배열의 모든 요소를 제로값으로 설정
			for j := 0; j < field.Len(); j++ {
				if field.Index(j).CanSet() {
					field.Index(j).Set(reflect.Zero(field.Index(j).Type()))
				}
			}
		case reflect.Struct:
			// time.Time 타입 처리
			if field.Type() == reflect.TypeOf(time.Time{}) {
				field.Set(reflect.Zero(field.Type()))
			} else {
				// 중첩 구조체는 재귀적으로 처리
				if field.CanAddr() {
					SetZero(field.Addr().Interface())
				}
			}
		default:
			// 기본적으로 제로값으로 설정
			field.Set(reflect.Zero(field.Type()))
		}
	}
}

// AddCounter는 객체의 int형 필드 값을 데이터베이스에서 증감시킵니다
func (r *Repository) AddCounter(obj interface{}, fieldName string, value int64, where map[string]interface{}) (int64, error) {
	if !r.IsConnected() {
		return -1, errors.New("데이터베이스 연결이 없습니다")
	}

	// 객체 타입 검증
	objType := reflect.TypeOf(obj)
	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}
	if objType.Kind() != reflect.Struct {
		return -1, errors.New("obj는 구조체 또는 구조체 포인터여야 합니다")
	}

	// 필드 존재성 및 타입 검증
	field, found := objType.FieldByName(fieldName)
	if !found {
		return -1, fmt.Errorf("필드 '%s'를 찾을 수 없습니다", fieldName)
	}

	// 비공개 필드 확인
	if field.PkgPath != "" {
		return -1, fmt.Errorf("필드 '%s'는 비공개 필드입니다", fieldName)
	}

	// int형 타입인지 확인
	fieldKind := field.Type.Kind()
	if fieldKind != reflect.Int && fieldKind != reflect.Int8 && fieldKind != reflect.Int16 &&
		fieldKind != reflect.Int32 && fieldKind != reflect.Int64 {
		return -1, fmt.Errorf("필드 '%s'는 int형이 아닙니다. 실제 타입: %s", fieldName, fieldKind.String())
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	// DB 필드 이름 가져오기
	dbFieldName := getDBFieldName(field)

	// UPDATE 쿼리 생성 (필드 값을 기존 값에 더하기)
	query := fmt.Sprintf("UPDATE `%s` SET `%s` = `%s` + ?", tableName, dbFieldName, dbFieldName)
	params := []interface{}{value}

	// WHERE 절 구성
	if len(where) > 0 {
		whereClause, whereParams := buildWhereClause(where)
		query += " WHERE " + whereClause
		params = append(params, whereParams...)
	}

	if EnableSQLLogging {
		logSQL(query, params)
	}

	// 쿼리 실행
	result, err := r.handler.Exec(query, params...)
	if err != nil {
		return -1, fmt.Errorf("카운터 업데이트 실패: %w", err)
	}

	// 영향받은 행 수 반환
	return result.RowsAffected()
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
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
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
			// default 태그가 있는지 확인하여 기본값 설정
			if defaultTag, ok := field.Tag.Lookup("default"); ok {
				if defaultTag == "CURRENT_TIMESTAMP" {
					return "datetime DEFAULT CURRENT_TIMESTAMP"
				} else if defaultTag == "CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" {
					return "datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"
				}
				return "datetime DEFAULT " + defaultTag
			}
			// 필드 이름이 UpdatedAt 혹은 updated_at인 경우 자동 업데이트 적용
			fieldName := field.Name
			dbFieldName := getDBFieldName(field)
			if fieldName == "UpdatedAt" || dbFieldName == "updated_at" {
				return "datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"
			}
			// Null 태그가 있는지 확인
			if _, nullOk := field.Tag.Lookup("Null"); nullOk {
				return "datetime NULL"
			}
			// 기본적으로 NOT NULL
			return "datetime NOT NULL"
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

// buildWhereClause는 WHERE 절을 구성하고 파라미터를 반환하는 공통 함수입니다
func buildWhereClause(where map[string]interface{}) (string, []interface{}) {
	if len(where) == 0 {
		return "", nil
	}

	conditions := make([]string, 0, len(where))
	params := make([]interface{}, 0, len(where)*2)

	for field, value := range where {
		// 특수 연산자 확인 (예: field__gt, field__like 등)
		parts := strings.Split(field, "__")
		fieldName := parts[0]
		operator := "="

		if len(parts) > 1 {
			// 특수 연산자 처리
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
			case "ilike":
				// MySQL에서는 ILIKE가 없으므로 LIKE + LOWER로 처리
				conditions = append(conditions, fmt.Sprintf("LOWER(`%s`) LIKE LOWER(?)", fieldName))
				params = append(params, value)
				continue
			case "in":
				// IN 연산자 처리
				if reflect.TypeOf(value).Kind() == reflect.Slice {
					s := reflect.ValueOf(value)
					if s.Len() > 0 {
						placeholders := make([]string, s.Len())
						for i := 0; i < s.Len(); i++ {
							placeholders[i] = "?"
							params = append(params, s.Index(i).Interface())
						}
						conditions = append(conditions, fmt.Sprintf("`%s` IN (%s)", fieldName, strings.Join(placeholders, ", ")))
						continue
					} else {
						// 빈 슬라이스인 경우 경고 로그 출력 후 조건 무시
						loghandle.Warn("buildWhereClause: 'in' 연산자는 비어있지 않은 슬라이스가 필요합니다. 필드: %s", fieldName)
						continue
					}
				} else {
					// 슬라이스가 아닌 경우 경고 로그 출력 후 조건 무시
					loghandle.Warn("buildWhereClause: 'in' 연산자는 슬라이스 타입이 필요합니다. 필드: %s, 값 타입: %T", fieldName, value)
					continue
				}
			case "notin":
				// NOT IN 연산자 처리
				if reflect.TypeOf(value).Kind() == reflect.Slice {
					s := reflect.ValueOf(value)
					if s.Len() > 0 {
						placeholders := make([]string, s.Len())
						for i := 0; i < s.Len(); i++ {
							placeholders[i] = "?"
							params = append(params, s.Index(i).Interface())
						}
						conditions = append(conditions, fmt.Sprintf("`%s` NOT IN (%s)", fieldName, strings.Join(placeholders, ", ")))
						continue
					} else {
						// 빈 슬라이스인 경우 경고 로그 출력 후 조건 무시
						loghandle.Warn("buildWhereClause: 'notin' 연산자는 비어있지 않은 슬라이스가 필요합니다. 필드: %s", fieldName)
						continue
					}
				} else {
					// 슬라이스가 아닌 경우 경고 로그 출력 후 조건 무시
					loghandle.Warn("buildWhereClause: 'notin' 연산자는 슬라이스 타입이 필요합니다. 필드: %s, 값 타입: %T", fieldName, value)
					continue
				}
			case "null":
				// IS NULL 또는 IS NOT NULL 처리
				if boolVal, ok := value.(bool); ok {
					if boolVal {
						conditions = append(conditions, fmt.Sprintf("`%s` IS NULL", fieldName))
					} else {
						conditions = append(conditions, fmt.Sprintf("`%s` IS NOT NULL", fieldName))
					}
					continue
				} else {
					// bool 타입이 아닌 경우 경고 로그 출력 후 조건 무시
					loghandle.Warn("buildWhereClause: 'null' 연산자는 boolean 값이 필요합니다. 필드: %s, 값: %v (타입: %T)", fieldName, value, value)
					continue
				}
			case "between":
				// BETWEEN 연산자 처리
				if reflect.TypeOf(value).Kind() == reflect.Slice {
					s := reflect.ValueOf(value)
					if s.Len() == 2 {
						conditions = append(conditions, fmt.Sprintf("`%s` BETWEEN ? AND ?", fieldName))
						params = append(params, s.Index(0).Interface(), s.Index(1).Interface())
						continue
					} else {
						// 슬라이스 길이가 2가 아닌 경우 경고 로그 출력 후 조건 무시
						loghandle.Warn("buildWhereClause: 'between' 연산자는 정확히 2개의 요소를 가진 슬라이스가 필요합니다. 필드: %s, 슬라이스 길이: %d", fieldName, s.Len())
						continue
					}
				} else {
					// 슬라이스가 아닌 경우 경고 로그 출력 후 조건 무시
					loghandle.Warn("buildWhereClause: 'between' 연산자는 슬라이스 타입이 필요합니다. 필드: %s, 값 타입: %T", fieldName, value)
					continue
				}
			case "contains":
				// 문자열 포함 검사 (LIKE %값%)
				conditions = append(conditions, fmt.Sprintf("`%s` LIKE ?", fieldName))
				params = append(params, fmt.Sprintf("%%%v%%", value))
				continue
			case "startswith":
				// 문자열 시작 검사 (LIKE 값%)
				conditions = append(conditions, fmt.Sprintf("`%s` LIKE ?", fieldName))
				params = append(params, fmt.Sprintf("%v%%", value))
				continue
			case "endswith":
				// 문자열 끝 검사 (LIKE %값)
				conditions = append(conditions, fmt.Sprintf("`%s` LIKE ?", fieldName))
				params = append(params, fmt.Sprintf("%%%v", value))
				continue
			case "regex":
				// 정규식 검사 (MySQL REGEXP)
				conditions = append(conditions, fmt.Sprintf("`%s` REGEXP ?", fieldName))
				params = append(params, value)
				continue
			}
		}

		// 기본 조건 추가
		conditions = append(conditions, fmt.Sprintf("`%s` %s ?", fieldName, operator))
		params = append(params, value)
	}

	return strings.Join(conditions, " AND "), params
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

	if v.Kind() != reflect.Struct {
		loghandle.Error("getFieldPointers: dest는 구조체 또는 구조체 포인터여야 합니다. 현재 타입: %v", v.Kind())
		return []interface{}{}
	}

	t := v.Type()
	pointers := make([]interface{}, 0, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// 비공개 필드 무시
		if field.PkgPath != "" {
			continue
		}

		fieldValue := v.Field(i)
		// 필드가 주소 지정 가능한지 확인
		if fieldValue.CanAddr() {
			pointers = append(pointers, fieldValue.Addr().Interface())
		} else {
			loghandle.Warn("getFieldPointers: 필드 '%s'는 주소 지정이 불가능합니다. 이 필드는 무시됩니다.", field.Name)
		}
	}

	return pointers
}

// CreateTableFromStruct는 구조체를 기반으로 테이블을 생성합니다
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
				logSQL(backupQuery, nil)
			}

			_, err := r.handler.Exec(backupQuery)
			if err != nil {
				return fmt.Errorf("백업 테이블 생성 실패: %w", err)
			}

			// 데이터 복사
			copyDataQuery := fmt.Sprintf("INSERT INTO `%s` SELECT * FROM `%s`", backupTableName, tableName)
			if EnableSQLLogging {
				logSQL(copyDataQuery, nil)
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
			logSQL(dropQuery, nil)
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
	var updatedAtField string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		dbFieldName := getDBFieldName(field)
		sqlType := getSQLType(field)
		nullable := ""
		defaultValue := ""

		// UpdatedAt 필드 검출
		if dbFieldName == "updated_at" && field.Type == reflect.TypeOf(time.Time{}) {
			updatedAtField = dbFieldName
		}

		// 기본적으로 NOT NULL
		if _, ok := field.Tag.Lookup("Null"); ok {
			nullable = " NULL"
		} else if sqlType != "datetime NOT NULL" &&
			!strings.Contains(sqlType, "DEFAULT CURRENT_TIMESTAMP") &&
			!strings.Contains(sqlType, "NULL") {
			nullable = " NOT NULL"
		}

		// 기본값 처리 (datetime 필드는 getSQLType에서 이미 처리됨)
		if !strings.Contains(sqlType, "DEFAULT") && !strings.Contains(sqlType, "NULL") {
			defaultValue = getDefaultValue(field)
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
		logSQL(query, nil)
	}

	// 쿼리 실행
	_, err := r.handler.Exec(query)
	if err != nil {
		return err
	}

	// updated_at 필드가 있으면 자동 업데이트를 위한 트리거 생성
	if updatedAtField != "" {
		// 기존 트리거가 있으면 삭제
		dropTriggerQuery := fmt.Sprintf("DROP TRIGGER IF EXISTS `%s_update_trigger`", tableName)
		if EnableSQLLogging {
			logSQL(dropTriggerQuery, nil)
		}
		_, err = r.handler.Exec(dropTriggerQuery)
		if err != nil {
			loghandle.Warn("트리거 삭제 실패, 계속 진행합니다: %v", err)
		}

		// 새 트리거 생성
		triggerQuery := fmt.Sprintf(`
			CREATE TRIGGER %s_update_trigger 
			BEFORE UPDATE ON %s 
			FOR EACH ROW 
			BEGIN
				SET NEW.%s = NOW();
			END
		`, tableName, tableName, updatedAtField)

		if EnableSQLLogging {
			logSQL(triggerQuery, nil)
		}

		_, err = r.handler.Exec(triggerQuery)
		if err != nil {
			loghandle.Warn("자동 업데이트 트리거 생성 실패, 계속 진행합니다: %v", err)
		} else {
			loghandle.Info("'%s' 테이블에 '%s' 필드 자동 업데이트 트리거 생성 성공", tableName, updatedAtField)
		}
	}

	return nil
}

// FindOneByQuery는 사용자 정의 SQL 쿼리로 단일 레코드를 조회하고 조회된 레코드 수를 반환합니다
// 예: count, err := repo.FindOneByQuery(&user, "SELECT * FROM users WHERE id = ?", 1)
func (r *Repository) FindOneByQuery(dest interface{}, query string, args ...interface{}) (int, error) {
	if !r.IsConnected() {
		return -1, errors.New("데이터베이스 연결이 없습니다")
	}

	if EnableSQLLogging {
		logSQL(query, args)
	}

	// 결과를 구조체로 스캔하기 위한 포인터 준비
	pointers := getFieldPointers(dest)
	if len(pointers) == 0 {
		return -1, errors.New("스캔할 필드가 없거나 dest가 적절한 구조체 타입이 아닙니다")
	}

	// 쿼리 실행
	row := r.handler.QueryRow(query, args...)

	// 결과를 구조체로 스캔
	if err := row.Scan(pointers...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// 레코드가 없는 경우 ErrNoRows 에러 반환 (Go 관습에 따라)
			return 0, sql.ErrNoRows
		}
		return -1, err
	}

	return 1, nil
}

// FindAllByQuery는 사용자 정의 SQL 쿼리로 여러 레코드를 조회하고 조회된 레코드 수를 반환합니다
func (r *Repository) FindAllByQuery(dest interface{}, query string, args ...interface{}) (int, error) {
	if !r.IsConnected() {
		return -1, errors.New("데이터베이스 연결이 없습니다")
	}

	// dest는 슬라이스 포인터여야 함
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return -1, errors.New("dest는 슬라이스 포인터여야 합니다")
	}

	if EnableSQLLogging {
		logSQL(query, args)
	}

	// 쿼리 실행
	rows, err := r.handler.Query(query, args...)
	if err != nil {
		return -1, err
	}
	defer rows.Close()

	// 슬라이스 요소 타입 가져오기
	sliceElemType := destValue.Elem().Type().Elem()
	isPointerElem := sliceElemType.Kind() == reflect.Ptr

	// 포인터 타입이면 대상 타입 가져오기
	structType := sliceElemType
	if isPointerElem {
		structType = sliceElemType.Elem()
	}

	// 결과를 담을 새 슬라이스 생성
	sliceValue := reflect.MakeSlice(destValue.Elem().Type(), 0, 0)

	// 컬럼 정보 가져오기
	columns, err := rows.Columns()
	if err != nil {
		return -1, fmt.Errorf("컬럼 정보 가져오기 실패: %w", err)
	}

	// 구조체 필드 매핑
	fieldMap := getStructFieldsMap(structType)

	count := 0
	// 각 결과 행 처리
	for rows.Next() {
		// 새 요소 생성 (포인터 또는 값 타입에 따라)
		var newElem reflect.Value
		if isPointerElem {
			// 포인터 타입이면 새 인스턴스 생성 후 포인터 생성
			newElem = reflect.New(structType)
		} else {
			// 값 타입이면 새 인스턴스 생성
			newElem = reflect.New(sliceElemType).Elem()
		}

		// 스캔 대상 준비
		scanValues := make([]interface{}, len(columns))
		for i, colName := range columns {
			fieldIndex, found := fieldMap[strings.ToLower(colName)]
			if found {
				field := newElem
				if isPointerElem {
					field = field.Elem()
				}

				fieldValue := field.Field(fieldIndex)
				fieldType := fieldValue.Type()

				// NULL 값을 처리하기 위한 스캔 대상 생성
				switch fieldType.Kind() {
				case reflect.String:
					scanValues[i] = &sql.NullString{}
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					scanValues[i] = &sql.NullInt64{}
				case reflect.Float32, reflect.Float64:
					scanValues[i] = &sql.NullFloat64{}
				case reflect.Bool:
					scanValues[i] = &sql.NullBool{}
				default:
					// 시간 타입 처리
					if fieldType == reflect.TypeOf(time.Time{}) {
						scanValues[i] = &sql.NullTime{}
					} else {
						// 기타 타입은 직접 주소 지정
						scanValues[i] = fieldValue.Addr().Interface()
					}
				}
			} else {
				// 컬럼이 구조체에 없으면 임시 변수에 스캔
				var v interface{}
				scanValues[i] = &v
			}
		}

		// 행 스캔
		if err := rows.Scan(scanValues...); err != nil {
			return count, fmt.Errorf("행 스캔 실패: %w", err)
		}

		// NULL 값을 Go 기본값으로 변환
		for i, colName := range columns {
			fieldIndex, found := fieldMap[strings.ToLower(colName)]
			if found {
				field := newElem
				if isPointerElem {
					field = field.Elem()
				}

				fieldValue := field.Field(fieldIndex)

				// NULL 스캔 값에서 실제 값으로 변환
				switch scanVal := scanValues[i].(type) {
				case *sql.NullString:
					if scanVal.Valid {
						fieldValue.SetString(scanVal.String)
					} else {
						fieldValue.SetString("")
					}
				case *sql.NullInt64:
					if scanVal.Valid {
						fieldValue.SetInt(scanVal.Int64)
					} else {
						fieldValue.SetInt(0)
					}
				case *sql.NullFloat64:
					if scanVal.Valid {
						fieldValue.SetFloat(scanVal.Float64)
					} else {
						fieldValue.SetFloat(0)
					}
				case *sql.NullBool:
					if scanVal.Valid {
						fieldValue.SetBool(scanVal.Bool)
					} else {
						fieldValue.SetBool(false)
					}
				case *sql.NullTime:
					if scanVal.Valid {
						// time.Time 타입은 직접 설정
						fieldValue.Set(reflect.ValueOf(scanVal.Time))
					} else {
						// 기본 시간 값으로 설정
						fieldValue.Set(reflect.ValueOf(time.Time{}))
					}
				}
			}
		}

		// 슬라이스에 추가
		if isPointerElem {
			sliceValue = reflect.Append(sliceValue, newElem)
		} else {
			sliceValue = reflect.Append(sliceValue, newElem)
		}

		count++
	}

	// 최종 슬라이스를 대상에 설정
	destValue.Elem().Set(sliceValue)

	if err := rows.Err(); err != nil {
		return -1, err
	}

	return count, nil
}

// getStructFieldsMap은 구조체의 필드를 컬럼명으로 인덱싱된 맵으로 반환합니다.
// 키는 소문자 DB 필드명이고 값은 구조체 필드 인덱스입니다.
func getStructFieldsMap(structType reflect.Type) map[string]int {
	// 포인터 타입인지 확인하고 필요시 역참조
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	if structType.Kind() != reflect.Struct {
		return nil
	}

	// 필드 이름 -> 인덱스 매핑
	fieldMap := make(map[string]int)
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.PkgPath != "" { // 비공개 필드 무시
			continue
		}
		// 데이터베이스 필드명 가져오기 (태그 사용 또는 필드명)
		dbFieldName := getDBFieldName(field)
		fieldMap[strings.ToLower(dbFieldName)] = i
	}
	return fieldMap
}

// InsertStruct는 구조체를 데이터베이스에 삽입합니다
func (r *Repository) InsertStruct(obj interface{}) (int64, error) {
	if !r.IsConnected() {
		return -1, errors.New("데이터베이스 연결이 없습니다")
	}

	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return -1, errors.New("객체는 구조체여야 합니다")
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

		// time.Time 필드가 zero value인 경우 처리
		if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
			timeValue, _ := fieldValue.Interface().(time.Time)
			if timeValue.IsZero() {
				// default 태그가 있는지 확인
				if defaultTag, ok := field.Tag.Lookup("default"); ok && defaultTag == "CURRENT_TIMESTAMP" {
					// 현재 시간으로 설정
					fieldValue = reflect.ValueOf(time.Now())
				} else if _, nullOk := field.Tag.Lookup("Null"); nullOk {
					// NULL 허용 필드는 스킵 (NULL 처리는 별도로 함)
					continue
				} else {
					// 기본값이 없고 NULL도 아닌 경우 현재 시간 사용
					fieldValue = reflect.ValueOf(time.Now())
				}
			}
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
		return -1, err
	}

	// 마지막 삽입 ID 반환
	id, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}

	return id, nil
}

// UpdateStruct는 구조체를 데이터베이스에서 업데이트합니다
func (r *Repository) UpdateStruct(obj interface{}, where map[string]interface{}) (int64, error) {
	if !r.IsConnected() {
		return -1, errors.New("데이터베이스 연결이 없습니다")
	}

	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return -1, errors.New("객체는 구조체여야 합니다")
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

		// time.Time 필드가 zero value인 경우 처리
		if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
			timeValue, _ := fieldValue.Interface().(time.Time)
			if timeValue.IsZero() {
				// omitempty 태그가 있으면 이 필드를 업데이트에서 제외
				if _, omitOk := field.Tag.Lookup("omitempty"); omitOk {
					continue
				}

				// default 태그가 있는지 확인
				if defaultTag, ok := field.Tag.Lookup("default"); ok && defaultTag == "CURRENT_TIMESTAMP" {
					// 현재 시간으로 설정
					fieldValue = reflect.ValueOf(time.Now())
				} else if _, nullOk := field.Tag.Lookup("Null"); nullOk {
					// NULL 허용 필드는 NULL로 설정
					setFields = append(setFields, fmt.Sprintf("`%s` = NULL", dbFieldName))
					continue
				} else {
					// 기본적으로 zero time은 업데이트에서 제외
					// 이것은 UPDATE와 UPSERT에서 다르게 처리해야 함
					continue
				}
			}
		} else if isZeroValue(fieldValue) {
			// 시간 이외의 필드도 omitempty가 있고 zero 값이면 업데이트에서 제외
			if _, omitOk := field.Tag.Lookup("omitempty"); omitOk {
				continue
			}
			// Null 태그가 있는 경우 NULL로 설정
			if _, nullOk := field.Tag.Lookup("Null"); nullOk {
				setFields = append(setFields, fmt.Sprintf("`%s` = NULL", dbFieldName))
				continue
			}
		}

		// 여기에 도달한 필드만 업데이트
		setFields = append(setFields, fmt.Sprintf("`%s` = ?", dbFieldName))
		params = append(params, fieldValue.Interface())
	}

	if len(setFields) == 0 {
		return -1, errors.New("업데이트할 필드가 없습니다")
	}

	// 기본 쿼리 생성
	query := fmt.Sprintf("UPDATE `%s` SET %s", tableName, strings.Join(setFields, ", "))

	// WHERE 절 추가
	if len(where) > 0 {
		whereClause, whereParams := buildWhereClause(where)
		query += " WHERE " + whereClause
		params = append(params, whereParams...)
	}

	if EnableSQLLogging {
		logSQL(query, params)
	}

	// 쿼리 실행
	result, err := r.handler.Exec(query, params...)
	if err != nil {
		return -1, err
	}

	// 영향받은 행 수 반환
	return result.RowsAffected()
}

// FindOptions는 Find 함수에서 사용할 옵션을 정의합니다
//
// 지원되는 WHERE 절 연산자:
// - 기본: "field": value (= 연산자)
// - 확장: "field__gt": value (>), "field__gte": value (>=), "field__lt": value (<), "field__lte": value (<=)
// - 확장: "field__ne": value (!=), "field__like": value (LIKE), "field__ilike": value (ILIKE)
// - 확장: "field__in": []value (IN), "field__notin": []value (NOT IN)
// - 확장: "field__null": true/false (IS NULL/IS NOT NULL)
// - 확장: "field__between": []value (BETWEEN), "field__contains": value (LIKE %value%)
// - 확장: "field__startswith": value (LIKE value%), "field__endswith": value (LIKE %value)
// - 확장: "field__regex": value (REGEXP)
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

// Find는 조건에 맞는 레코드를 조회하고 조회된 레코드 수를 반환합니다
func (r *Repository) Find(dest interface{}, options *FindOptions) (int, error) {
	if !r.IsConnected() {
		return -1, errors.New("데이터베이스 연결이 없습니다")
	}

	// dest 검증
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr {
		return -1, errors.New("dest는 포인터여야 합니다")
	}

	elem := v.Elem()
	var structType reflect.Type
	var isSingle bool

	// 단일 구조체인지 슬라이스인지 확인
	if elem.Kind() == reflect.Struct {
		structType = elem.Type()
		isSingle = true
	} else if elem.Kind() == reflect.Slice {
		sliceType := elem.Type()
		elemType := sliceType.Elem()

		// 슬라이스 요소가 포인터인지 확인
		if elemType.Kind() == reflect.Ptr {
			structType = elemType.Elem()
		} else {
			structType = elemType
		}

		if structType.Kind() != reflect.Struct {
			return -1, errors.New("슬라이스 요소는 구조체여야 합니다")
		}
		isSingle = false
	} else {
		return -1, errors.New("dest는 구조체 포인터 또는 구조체 슬라이스 포인터여야 합니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(reflect.New(structType).Elem().Interface())

	// 쿼리 생성
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT ")

	// DISTINCT 처리
	if options != nil && options.Distinct {
		queryBuilder.WriteString("DISTINCT ")
	}

	// 컬럼 선택
	if options != nil && len(options.Columns) > 0 {
		columns := make([]string, len(options.Columns))
		for i, col := range options.Columns {
			columns[i] = "`" + col + "`"
		}
		queryBuilder.WriteString(strings.Join(columns, ", "))
	} else {
		// 모든 필드 선택
		columns := make([]string, 0, structType.NumField())
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			if field.PkgPath == "" { // 공개 필드만 선택
				dbFieldName := getDBFieldName(field)
				columns = append(columns, "`"+dbFieldName+"`")
			}
		}
		queryBuilder.WriteString(strings.Join(columns, ", "))
	}

	queryBuilder.WriteString(fmt.Sprintf(" FROM `%s`", tableName))

	// WHERE 절 구성
	var params []interface{}
	if options != nil && len(options.Where) > 0 {
		queryBuilder.WriteString(" WHERE ")
		whereClause, whereParams := buildWhereClause(options.Where)
		queryBuilder.WriteString(whereClause)
		params = whereParams
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
			havingClause, havingParams := buildWhereClause(options.HavingCond)
			queryBuilder.WriteString(havingClause)
			params = append(params, havingParams...)
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
		rows, err := r.handler.Query(query, params...)
		if err != nil {
			return -1, err
		}
		defer rows.Close()

		if !rows.Next() {
			if err := rows.Err(); err != nil {
				return -1, err
			}
			// 빈 결과는 정상 상황으로 처리 (에러 없이 Count: 0 반환)
			return 0, nil
		}

		// 컬럼 정보 가져오기
		columns, err := rows.Columns()
		if err != nil {
			return -1, fmt.Errorf("컬럼 정보 가져오기 실패: %w", err)
		}

		// NULL 처리를 포함한 스캔 수행
		err = scanWithNullCheck(rows, columns, dest)
		if err != nil {
			return -1, err
		}

		return 1, nil
	}

	// 여러 레코드 조회
	rows, err := r.handler.Query(query, params...)
	if err != nil {
		return -1, err
	}
	defer rows.Close()

	// 슬라이스 요소 타입 및 변환 방식 결정
	sliceElemType := elem.Type().Elem() // 슬라이스의 요소 타입
	isPointerElem := sliceElemType.Kind() == reflect.Ptr

	// 결과를 담을 새 슬라이스 생성
	sliceValue := reflect.MakeSlice(elem.Type(), 0, 0)

	// 컬럼 정보 가져오기
	columns, err := rows.Columns()
	if err != nil {
		return -1, fmt.Errorf("컬럼 정보 가져오기 실패: %w", err)
	}

	// 구조체 필드 매핑
	fieldMap := getStructFieldsMap(structType)

	count := 0
	// 각 결과 행 처리
	for rows.Next() {
		// 새 요소 생성 (포인터 또는 값 타입에 따라)
		var newElem reflect.Value
		if isPointerElem {
			// 포인터 타입이면 새 인스턴스 생성 후 포인터 생성
			newElem = reflect.New(structType)
		} else {
			// 값 타입이면 새 인스턴스 생성
			newElem = reflect.New(sliceElemType).Elem()
		}

		// 스캔 대상 준비
		scanValues := make([]interface{}, len(columns))
		tempValues := make([]interface{}, len(columns))
		for i, colName := range columns {
			fieldIndex, found := fieldMap[strings.ToLower(colName)]
			if found {
				field := newElem
				if isPointerElem {
					field = field.Elem()
				} else if field.Kind() == reflect.Struct {
					// 값 타입인 경우 필드에 직접 접근
				}

				fieldValue := field.Field(fieldIndex)

				// 필드가 주소를 가질 수 있으면 직접 스캔
				if fieldValue.CanAddr() {
					scanValues[i] = fieldValue.Addr().Interface()
				} else {
					// 임시 변수에 스캔 후 나중에 복사
					tempVal := reflect.New(fieldValue.Type())
					tempValues[i] = fieldValue
					scanValues[i] = tempVal.Interface()
				}
			} else {
				// 컬럼이 구조체에 없으면 임시 변수에 스캔
				var v interface{}
				scanValues[i] = &v
			}
		}

		// 행 스캔
		if err := rows.Scan(scanValues...); err != nil {
			return -1, fmt.Errorf("행 스캔 실패: %w", err)
		}

		// 임시 변수에서 복사해야 하는 경우 처리
		for i, tempVal := range tempValues {
			if tempVal != nil {
				// 임시 변수에서 대상 필드로 값 복사
				dest := tempVal.(reflect.Value)
				src := reflect.ValueOf(scanValues[i]).Elem().Elem()
				dest.Set(src)
			}
		}

		// 슬라이스에 추가
		if isPointerElem {
			sliceValue = reflect.Append(sliceValue, newElem)
		} else {
			sliceValue = reflect.Append(sliceValue, newElem)
		}

		count++
	}

	// 최종 슬라이스를 대상에 설정
	elem.Set(sliceValue)

	if err := rows.Err(); err != nil {
		return -1, err
	}

	return count, nil
}

// scanWithNullCheck는 NULL 값을 적절히 처리하는 스캔 함수입니다
func scanWithNullCheck(rows *sql.Rows, columns []string, dest interface{}) error {
	// 슬라이스 요소 타입 및 변환 방식 결정
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr {
		return errors.New("대상은 포인터여야 합니다")
	}
	v = v.Elem()

	// 구조체 정보 가져오기
	var structType reflect.Type
	if v.Kind() == reflect.Struct {
		structType = v.Type()
	} else {
		return errors.New("대상은 구조체 포인터여야 합니다")
	}

	// 구조체 필드 매핑
	fieldMap := getStructFieldsMap(structType)

	// 스캔 대상 준비
	scanValues := make([]interface{}, len(columns))
	for i, colName := range columns {
		fieldIndex, found := fieldMap[strings.ToLower(colName)]
		if found {
			field := v.Field(fieldIndex)
			// NULL 값을 처리하기 위한 스캔 대상 생성
			switch field.Kind() {
			case reflect.String:
				scanValues[i] = &sql.NullString{}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				scanValues[i] = &sql.NullInt64{}
			case reflect.Float32, reflect.Float64:
				scanValues[i] = &sql.NullFloat64{}
			case reflect.Bool:
				scanValues[i] = &sql.NullBool{}
			default:
				// 시간 타입 처리
				if field.Type() == reflect.TypeOf(time.Time{}) {
					scanValues[i] = &sql.NullTime{}
				} else {
					// 기타 타입은 직접 주소 지정
					scanValues[i] = field.Addr().Interface()
				}
			}
		} else {
			// 컬럼이 구조체에 없으면 임시 변수에 스캔
			var v interface{}
			scanValues[i] = &v
		}
	}

	// 행 스캔
	if err := rows.Scan(scanValues...); err != nil {
		return fmt.Errorf("행 스캔 실패: %w", err)
	}

	// NULL 값 처리 및 필드에 값 설정
	for i, colName := range columns {
		fieldIndex, found := fieldMap[strings.ToLower(colName)]
		if found {
			field := v.Field(fieldIndex)

			// NULL 스캔 값에서 실제 값으로 변환
			switch scanVal := scanValues[i].(type) {
			case *sql.NullString:
				if scanVal.Valid {
					field.SetString(scanVal.String)
				} else {
					field.SetString("")
				}
			case *sql.NullInt64:
				if scanVal.Valid {
					field.SetInt(scanVal.Int64)
				} else {
					field.SetInt(0)
				}
			case *sql.NullFloat64:
				if scanVal.Valid {
					field.SetFloat(scanVal.Float64)
				} else {
					field.SetFloat(0)
				}
			case *sql.NullBool:
				if scanVal.Valid {
					field.SetBool(scanVal.Bool)
				} else {
					field.SetBool(false)
				}
			case *sql.NullTime:
				if scanVal.Valid {
					// time.Time 타입은 직접 설정
					field.Set(reflect.ValueOf(scanVal.Time))
				} else {
					// 기본 시간 값으로 설정
					field.Set(reflect.ValueOf(time.Time{}))
				}
			}
		}
	}

	return nil
}

// FindOne은 단일 레코드를 조회하고 조회된 레코드 수를 반환합니다
func (r *Repository) FindOne(dest interface{}, where map[string]interface{}) (int, error) {
	options := &FindOptions{
		Where: where,
		// Limit: 1,
	}

	// 테이블 이름 가져오기
	t := reflect.TypeOf(dest)
	if t.Kind() != reflect.Ptr {
		return -1, errors.New("dest는 포인터여야 합니다")
	}
	t = t.Elem()
	if t.Kind() != reflect.Struct {
		return -1, errors.New("dest는 구조체 포인터여야 합니다")
	}

	tableName := getTableName(reflect.New(t).Elem().Interface())

	// 쿼리 생성
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT ")

	// 컬럼 선택
	if len(options.Columns) > 0 {
		columns := make([]string, len(options.Columns))
		for i, col := range options.Columns {
			columns[i] = "`" + col + "`"
		}
		queryBuilder.WriteString(strings.Join(columns, ", "))
	} else {
		// 모든 필드 선택
		columns := make([]string, 0, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath == "" { // 공개 필드만 선택
				dbFieldName := getDBFieldName(field)
				columns = append(columns, "`"+dbFieldName+"`")
			}
		}
		queryBuilder.WriteString(strings.Join(columns, ", "))
	}

	queryBuilder.WriteString(fmt.Sprintf(" FROM `%s`", tableName))

	// WHERE 절 구성
	var params []interface{}
	if len(options.Where) > 0 {
		queryBuilder.WriteString(" WHERE ")
		whereClause, whereParams := buildWhereClause(options.Where)
		queryBuilder.WriteString(whereClause)
		params = whereParams
	}

	// LIMIT 추가
	queryBuilder.WriteString(" LIMIT 1")

	query := queryBuilder.String()

	if EnableSQLLogging {
		logSQL(query, params)
	}

	// 쿼리 실행
	rows, err := r.handler.Query(query, params...)
	if err != nil {
		return -1, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return -1, err
		}
		// 레코드가 없는 경우 ErrNoRows 에러 반환 (Go 관습에 따라)
		return 0, sql.ErrNoRows
	}

	// 컬럼 정보 가져오기
	columns, err := rows.Columns()
	if err != nil {
		return -1, fmt.Errorf("컬럼 정보 가져오기 실패: %w", err)
	}

	// NULL 처리를 포함한 스캔 수행
	err = scanWithNullCheck(rows, columns, dest)
	if err != nil {
		return -1, err
	}

	return 1, nil
}

// FindAll은 여러 레코드를 조회하고 조회된 레코드 수를 반환합니다
func (r *Repository) FindAll(dest interface{}, options *FindOptions) (int, error) {
	return r.Find(dest, options)
}

// Count는 조건에 맞는 레코드 수를 반환합니다
func (r *Repository) Count(obj interface{}, where map[string]interface{}) (int64, error) {
	if !r.IsConnected() {
		return -1, errors.New("데이터베이스 연결이 없습니다")
	}

	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return -1, errors.New("객체는 구조체여야 합니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	// 쿼리 조합
	query := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", tableName)
	var params []interface{}

	// WHERE 절 추가
	if len(where) > 0 {
		whereClause, whereParams := buildWhereClause(where)
		query += " WHERE " + whereClause
		params = whereParams
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
		return -1, err
	}

	return count, nil
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

// UpdateFields updates specific fields in the database table based on the provided field map and conditions
func (r *Repository) UpdateFields(tableName string, fields map[string]interface{}, where map[string]interface{}) (int64, error) {
	if !r.IsConnected() {
		return 0, errors.New("데이터베이스 연결이 없습니다")
	}

	// UPDATE 쿼리 구성
	var setFields []string
	var params []interface{}

	for field, value := range fields {
		setFields = append(setFields, fmt.Sprintf("`%s` = ?", field))
		params = append(params, value)
	}

	query := fmt.Sprintf("UPDATE `%s` SET %s", tableName, strings.Join(setFields, ", "))

	// WHERE 절 구성
	if len(where) > 0 {
		whereClause, whereParams := buildWhereClause(where)
		query += " WHERE " + whereClause
		params = append(params, whereParams...)
	}

	// 로깅
	if EnableSQLLogging {
		logSQL(query, params)
	}

	// 쿼리 실행
	result, err := r.handler.Exec(query, params...)
	if err != nil {
		return -1, err
	}

	// 영향받은 행 수 반환
	return result.RowsAffected()
}

// UpdateFieldsByStruct updates specific fields in the database table based on the provided struct and conditions
func (r *Repository) UpdateFieldsByStruct(obj interface{}, updateFields []string, where map[string]interface{}) (int64, error) {
	if !r.IsConnected() {
		return -1, errors.New("데이터베이스 연결이 없습니다")
	}

	// 객체 타입 검증
	objType := reflect.TypeOf(obj)
	objValue := reflect.ValueOf(obj)
	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
		objValue = objValue.Elem()
	}
	if objType.Kind() != reflect.Struct {
		return -1, errors.New("obj는 구조체 또는 구조체 포인터여야 합니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	// UPDATE 쿼리 구성
	var setFields []string
	var params []interface{}

	// 지정된 필드만 업데이트
	for _, fieldName := range updateFields {
		field, found := objType.FieldByName(fieldName)
		if !found {
			continue
		}

		// 비공개 필드 무시
		if field.PkgPath != "" {
			continue
		}

		dbFieldName := getDBFieldName(field)
		fieldValue := objValue.FieldByName(fieldName)

		setFields = append(setFields, fmt.Sprintf("`%s` = ?", dbFieldName))
		params = append(params, fieldValue.Interface())
	}

	if len(setFields) == 0 {
		return -1, errors.New("업데이트할 필드가 없습니다")
	}

	query := fmt.Sprintf("UPDATE `%s` SET %s", tableName, strings.Join(setFields, ", "))

	// WHERE 절 구성
	if len(where) > 0 {
		whereClause, whereParams := buildWhereClause(where)
		query += " WHERE " + whereClause
		params = append(params, whereParams...)
	}

	// 로깅
	if EnableSQLLogging {
		logSQL(query, params)
	}

	// 쿼리 실행
	result, err := r.handler.Exec(query, params...)
	if err != nil {
		return -1, err
	}

	// 영향받은 행 수 반환
	return result.RowsAffected()
}

// UpsertStruct inserts a new record or updates an existing one based on the primary key
func (r *Repository) UpsertStruct(obj interface{}) (int64, error) {
	if r.handler == nil {
		return -1, errors.New("데이터베이스 연결이 설정되지 않았습니다")
	}

	// 객체 타입 검증
	objType := reflect.TypeOf(obj)
	objValue := reflect.ValueOf(obj)
	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
		objValue = objValue.Elem()
	}
	if objType.Kind() != reflect.Struct {
		return -1, errors.New("obj는 구조체 또는 구조체 포인터여야 합니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	// 필드 정보 수집
	var columns []string
	var values []interface{}
	var updateClauses []string

	pkField, hasPK, isAuto := getPrimaryKeyField(objType)
	var pkValue interface{}

	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		fieldValue := objValue.Field(i)

		// 비공개 필드 무시
		if field.PkgPath != "" {
			continue
		}

		dbFieldName := getDBFieldName(field)

		// 자동 증가 PK는 값이 0인 경우 INSERT 부분에서 제외
		if isAuto && dbFieldName == pkField && isZeroValue(fieldValue) {
			continue
		}

		// 비어있지 않은 필드만 처리
		if !isZeroValue(fieldValue) {
			// PK 값 저장
			if hasPK && dbFieldName == pkField {
				pkValue = fieldValue.Interface()
			}

			columns = append(columns, "`"+dbFieldName+"`")
			values = append(values, fieldValue.Interface())
			updateClauses = append(updateClauses, fmt.Sprintf("`%s` = VALUES(`%s`)", dbFieldName, dbFieldName))
		}
	}

	if len(columns) == 0 {
		return -1, errors.New("삽입할 필드가 없습니다")
	}

	// 쿼리 구성
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(
		"INSERT INTO `%s` (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(updateClauses, ", "),
	)

	// 로깅
	if EnableSQLLogging {
		logSQL(query, values)
	}

	// 쿼리 실행
	result, err := r.handler.Exec(query, values...)
	if err != nil {
		return -1, err
	}

	// 영향받은 행 수 또는 마지막 삽입 ID 반환
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return -1, err
	}

	// 행이 영향을 받지 않았다면 오류 반환
	if rowsAffected == 0 {
		return 0, sql.ErrNoRows
	}

	// 새 레코드가 삽입된 경우 LastInsertId를, 그렇지 않으면 행 수를 반환
	lastInsertID, err := result.LastInsertId()
	if err == nil && lastInsertID > 0 && isAuto {
		return lastInsertID, nil
	}

	// PK 값이 있으면 반환 (업데이트된 경우)
	if hasPK && pkValue != nil {
		switch v := pkValue.(type) {
		case int64:
			return v, nil
		case int:
			return int64(v), nil
		case uint64:
			return int64(v), nil
		case uint:
			return int64(v), nil
		}
	}

	return rowsAffected, nil
}

// Batch는 여러 데이터베이스 작업을 한 번에 처리하는 배치 작업을 지원합니다
type Batch struct {
	repo      *Repository
	queries   []string
	params    [][]interface{}
	committed bool
}

// NewBatch는 새로운 배치 작업을 생성합니다
func (r *Repository) NewBatch() (*Batch, error) {
	if !r.IsConnected() {
		return nil, errors.New("데이터베이스 연결이 없습니다")
	}

	return &Batch{
		repo:      r,
		queries:   []string{},
		params:    [][]interface{}{},
		committed: false,
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

	// 배치에 추가
	b.queries = append(b.queries, query)
	b.params = append(b.params, params)

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

		setFields = append(setFields, fmt.Sprintf("`%s` = ?", dbFieldName))
		params = append(params, fieldValue.Interface())
	}

	// 기본 쿼리 생성
	query := fmt.Sprintf("UPDATE `%s` SET %s", tableName, strings.Join(setFields, ", "))

	// WHERE 절 구성
	if len(where) > 0 {
		whereClause, whereParams := buildWhereClause(where)
		query += " WHERE " + whereClause
		params = append(params, whereParams...) // SET 파라미터 뒤에 WHERE 파라미터 추가
	}

	// 배치에 추가
	b.queries = append(b.queries, query)
	b.params = append(b.params, params)

	return nil
}

// AddDelete는 배치에 DELETE 작업을 추가합니다
func (b *Batch) AddDelete(obj interface{}, where map[string]interface{}) error {
	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	// 쿼리 구성
	query := fmt.Sprintf("DELETE FROM `%s`", tableName)
	var params []interface{}

	// WHERE 절 추가
	if len(where) > 0 {
		whereClause, whereParams := buildWhereClause(where)
		query += " WHERE " + whereClause
		params = whereParams
	} else {
		// WHERE 조건 없는 삭제 방지 (필요 시 주석 해제)
		// return errors.New("AddDelete: WHERE 조건 없는 삭제는 위험합니다.")
	}

	// 배치에 추가
	b.queries = append(b.queries, query)
	b.params = append(b.params, params)

	return nil
}

// Execute는 배치의 모든 작업을 하나의 트랜잭션으로 실행합니다
func (b *Batch) Execute() (int64, error) {
	if b.committed {
		return -1, errors.New("배치가 이미 커밋되었습니다")
	}
	if len(b.queries) == 0 {
		loghandle.Info("Batch.Execute: 실행할 작업이 없습니다.")
		return 0, nil
	}

	if b.repo == nil || !b.repo.IsConnected() {
		return -1, errors.New("Batch.Execute: 유효한 데이터베이스 연결이 없습니다.")
	}

	// 트랜잭션 시작
	loghandle.Info("Batch: 트랜잭션 시작 (총 %d개 작업)", len(b.queries))
	tx, err := b.repo.handler.Begin() // handler의 Begin 메서드 사용
	if err != nil {
		loghandle.Error("Batch: 트랜잭션 시작 실패: %v", err)
		return 0, fmt.Errorf("트랜잭션 시작 실패: %w", err)
	}

	// defer를 사용하여 롤백 처리
	shouldRollback := true
	defer func() {
		if shouldRollback {
			loghandle.Warn("Batch: 오류 발생으로 트랜잭션 롤백 시도...")
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				loghandle.Error("Batch: 트랜잭션 롤백 중 추가 오류 발생: %v", rollbackErr)
			} else {
				loghandle.Warn("Batch: 트랜잭션 롤백 완료.")
			}
		}
	}()

	// 모든 쿼리 실행
	var totalAffected int64 = 0
	for i, query := range b.queries {
		logSQL(query, b.params[i]) // 각 쿼리 로깅

		result, err := tx.Exec(query, b.params[i]...)
		if err != nil {
			loghandle.Error("Batch: 작업 %d/%d 실행 실패: %v", i+1, len(b.queries), err)
			// 실패 시 쿼리와 파라미터 다시 로깅
			loghandle.Error("Failed Query: %s", query)
			loghandle.Error("Failed Params: %v", b.params[i])
			// shouldRollback = true (defer가 롤백 처리)
			return 0, fmt.Errorf("배치 작업 %d 실행 실패: %w", i+1, err)
		}

		affected, err := result.RowsAffected()
		if err == nil {
			totalAffected += affected
			loghandle.Debug("Batch: 작업 %d/%d 실행 성공 (Affected: %d)", i+1, len(b.queries), affected)
		} else {
			loghandle.Warn("Batch: 작업 %d/%d 영향 받은 행 수 확인 실패: %v", i+1, len(b.queries), err)
		}
	}

	// 트랜잭션 커밋
	loghandle.Info("Batch: 모든 작업 성공, 트랜잭션 커밋 시도...")
	err = tx.Commit()
	if err != nil {
		loghandle.Error("Batch: 트랜잭션 커밋 실패: %v", err)
		// shouldRollback = true (defer가 롤백 처리)
		return 0, fmt.Errorf("트랜잭션 커밋 실패: %w", err)
	}

	shouldRollback = false // 커밋 성공, 롤백 방지
	loghandle.Info("Batch: 트랜잭션 커밋 완료. (총 영향 받은 행: %d)", totalAffected)
	b.committed = true
	return totalAffected, nil
}

// Rollback은 배치의 모든 작업을 롤백합니다 (Execute 전에만 의미 있음)
func (b *Batch) Rollback() error {
	if b.committed {
		return errors.New("배치가 이미 커밋되었습니다")
	}
	// 실제 롤백은 Execute 내부의 defer에서 처리됨
	// 이 메서드는 주로 Execute 전에 작업을 취소하고 싶을 때 사용될 수 있음 (현재 구현은 비어있음)
	loghandle.Warn("Batch.Rollback: 배치가 아직 실행되지 않았으므로 실제 롤백 작업은 수행되지 않습니다.")
	return nil
}

// UpdateNonZero는 제로값이 아닌 필드만 업데이트합니다
func (r *Repository) UpdateNonZero(obj interface{}, where map[string]interface{}, original ...interface{}) (int64, error) {
	if !r.IsConnected() {
		return -1, errors.New("데이터베이스 연결이 없습니다")
	}

	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return -1, errors.New("객체는 구조체여야 합니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	// 필드와 값 추출
	var setFields []string
	var params []interface{}

	pkField, _, _ := getPrimaryKeyField(t)

	// 원본 값이 제공된 경우 (변경된 필드만 업데이트)
	var originalValue reflect.Value
	if len(original) > 0 && original[0] != nil {
		originalValue = reflect.ValueOf(original[0])
		if originalValue.Kind() == reflect.Ptr {
			originalValue = originalValue.Elem()
		}
	}

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

		// auto 태그가 있으면 자동 업데이트에서 제외
		if _, autoOk := field.Tag.Lookup("auto"); autoOk {
			continue
		}

		// Zero 값이 아닌 필드만 업데이트
		if !isZeroValue(fieldValue) {
			// 원본이 제공된 경우 값이 변경된 필드만 업데이트
			if originalValue.IsValid() {
				originalField := originalValue.FieldByName(field.Name)
				if originalField.IsValid() && reflect.DeepEqual(fieldValue.Interface(), originalField.Interface()) {
					continue
				}
			}

			setFields = append(setFields, fmt.Sprintf("`%s` = ?", dbFieldName))
			params = append(params, fieldValue.Interface())
		}
	}

	if len(setFields) == 0 {
		return -1, errors.New("업데이트할 필드가 없습니다")
	}

	// 기본 쿼리 생성
	query := fmt.Sprintf("UPDATE `%s` SET %s", tableName, strings.Join(setFields, ", "))

	// WHERE 절 추가
	if len(where) > 0 {
		whereClause, whereParams := buildWhereClause(where)
		query += " WHERE " + whereClause
		params = append(params, whereParams...)
	}

	if EnableSQLLogging {
		logSQL(query, params)
	}

	// 쿼리 실행
	result, err := r.handler.Exec(query, params...)
	if err != nil {
		return -1, err
	}

	// 영향받은 행 수 반환
	return result.RowsAffected()
}

// DeleteStruct는 구조체에 해당하는 레코드를 삭제합니다
func (r *Repository) DeleteStruct(obj interface{}, where map[string]interface{}) (int64, error) {
	if !r.IsConnected() {
		return -1, errors.New("데이터베이스 연결이 없습니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	// 쿼리 구성
	query := fmt.Sprintf("DELETE FROM `%s`", tableName)
	var params []interface{}

	// WHERE 절 추가
	if len(where) > 0 {
		whereClause, whereParams := buildWhereClause(where)
		query += " WHERE " + whereClause
		params = whereParams
	}

	if EnableSQLLogging {
		logSQL(query, params)
	}

	// 쿼리 실행
	result, err := r.handler.Exec(query, params...)
	if err != nil {
		return -1, err
	}

	// 영향받은 행 수 반환
	return result.RowsAffected()
}

// isTimestampField checks if the field is a timestamp field
// func isTimestampField(field reflect.StructField) bool {
// 	return field.Type == reflect.TypeOf(time.Time{})
// }

// UpsertNonZero는 제로값이 아닌 필드만 사용하여 새 레코드를 삽입하거나 기존 레코드를 업데이트합니다
func (r *Repository) UpsertNonZero(obj interface{}) (int64, error) {
	if !r.IsConnected() {
		return -1, errors.New("데이터베이스 연결이 없습니다")
	}

	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return -1, errors.New("객체는 구조체여야 합니다")
	}

	// 테이블 이름 가져오기
	tableName := getTableName(obj)

	// 필드와 값 추출
	var columns []string
	var placeholders []string
	var updateClauses []string
	var params []interface{}

	pkField, hasPK, isAuto := getPrimaryKeyField(t)
	var pkValue interface{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// 비공개 필드 무시
		if field.PkgPath != "" {
			continue
		}

		dbFieldName := getDBFieldName(field)

		// 자동 증가 PK는 값이 0인 경우 INSERT 부분에서 제외
		if isAuto && dbFieldName == pkField && isZeroValue(fieldValue) {
			continue
		}

		// 비어있지 않은 필드만 처리
		if !isZeroValue(fieldValue) {
			// PK 값 저장
			if hasPK && dbFieldName == pkField {
				pkValue = fieldValue.Interface()
			}

			columns = append(columns, "`"+dbFieldName+"`")
			placeholders = append(placeholders, "?")
			params = append(params, fieldValue.Interface())
			updateClauses = append(updateClauses, fmt.Sprintf("`%s` = VALUES(`%s`)", dbFieldName, dbFieldName))
		}
	}

	if len(columns) == 0 {
		return -1, errors.New("삽입할 필드가 없습니다")
	}

	// 쿼리 생성
	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(updateClauses, ", "))

	if EnableSQLLogging {
		logSQL(query, params)
	}

	// 쿼리 실행
	result, err := r.handler.Exec(query, params...)
	if err != nil {
		return -1, err
	}

	// 영향받은 행 수 또는 마지막 삽입 ID 반환
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return -1, err
	}

	// 행이 영향을 받지 않았다면 오류 반환
	if rowsAffected == 0 {
		return 0, sql.ErrNoRows
	}

	// 새 레코드가 삽입된 경우 LastInsertId를, 그렇지 않으면 행 수를 반환
	lastInsertID, err := result.LastInsertId()
	if err == nil && lastInsertID > 0 && isAuto {
		return lastInsertID, nil
	}

	// PK 값이 있으면 반환 (업데이트된 경우)
	if hasPK && pkValue != nil {
		switch v := pkValue.(type) {
		case int64:
			return v, nil
		case int:
			return int64(v), nil
		case uint64:
			return int64(v), nil
		case uint:
			return int64(v), nil
		}
	}

	return rowsAffected, nil
}

// InsertList는 트랜잭션 내에서 여러 구조체 레코드를 벌크 INSERT 합니다.
// items는 동일한 구조체 타입의 슬라이스여야 합니다 (포인터 권장).
// tx 인자로 전달된 트랜잭션 내에서 실행됩니다.
func InsertList(tx *sql.Tx, items []interface{}) error {
	if len(items) == 0 {
		loghandle.Debug("InsertList: 삽입할 항목이 없습니다.")
		return nil // 삽입할 항목이 없으면 성공 처리
	}

	// 첫 번째 항목을 기반으로 타입 및 테이블 정보 가져오기
	firstItem := items[0]
	itemType := reflect.TypeOf(firstItem)
	isPointer := itemType.Kind() == reflect.Ptr
	if isPointer {
		itemType = itemType.Elem() // 포인터면 실제 타입 가져오기
	}
	if itemType.Kind() != reflect.Struct {
		return errors.New("InsertList: items 슬라이스는 구조체 또는 구조체 포인터를 포함해야 합니다")
	}

	tableName := getTableName(firstItem) // 테이블 이름 가져오기 (포인터/값 상관없이 동작)
	loghandle.Debug("InsertList: 테이블 '%s'에 %d개 항목 삽입 시작", tableName, len(items))

	pkField, _, isAuto := getPrimaryKeyField(itemType) // 타입 정보로 PK 확인

	// 컬럼 목록 생성 (자동 증가 PK 제외)
	var columns []string
	var fieldIndices []int // 값 추출을 위한 필드 인덱스 저장
	for i := 0; i < itemType.NumField(); i++ {
		field := itemType.Field(i)
		if field.PkgPath != "" { // 비공개 필드 무시
			continue
		}
		dbFieldName := getDBFieldName(field)
		// 자동 증가 PK 필드는 INSERT 컬럼 목록에서 제외
		if isAuto && dbFieldName == pkField {
			continue
		}
		columns = append(columns, "`"+dbFieldName+"`")
		fieldIndices = append(fieldIndices, i) // 해당 컬럼 값의 필드 인덱스 저장
	}

	if len(columns) == 0 {
		return fmt.Errorf("InsertList: 테이블 '%s'에 삽입할 컬럼이 없습니다 (PK만 있는 경우?)", tableName)
	}
	loghandle.Debug("InsertList: 테이블 '%s' 삽입 컬럼: %v", tableName, columns)

	// VALUES (?, ?, ...), (?, ?, ...), ... 부분 생성 및 파라미터 준비
	valueStrings := make([]string, 0, len(items))
	allParams := make([]interface{}, 0, len(items)*len(columns))
	placeholder := "(?" + strings.Repeat(",?", len(columns)-1) + ")" // (?,?,?) 형태

	for idx, item := range items {
		v := reflect.ValueOf(item)
		if v.Kind() == reflect.Ptr {
			// 타입 일관성 검사 (포인터 타입이어야 함)
			if !isPointer || v.Type().Elem() != itemType {
				return fmt.Errorf("InsertList: 슬라이스의 %d번째 항목 타입(%v)이 첫 번째 항목 타입(*%v)과 다릅니다", idx, v.Type(), itemType)
			}
			v = v.Elem() // 포인터면 실제 값으로
		} else {
			// 타입 일관성 검사 (값 타입이어야 함)
			if isPointer || v.Type() != itemType {
				return fmt.Errorf("InsertList: 슬라이스의 %d번째 항목 타입(%v)이 첫 번째 항목 타입(%v)과 다릅니다", idx, v.Type(), itemType)
			}
		}

		// 구조체가 유효한지 확인
		if !v.IsValid() || v.Kind() != reflect.Struct {
			return fmt.Errorf("InsertList: 슬라이스의 %d번째 항목이 유효한 구조체가 아닙니다", idx)
		}

		valueStrings = append(valueStrings, placeholder) // 각 행에 대한 플레이스홀더 추가
		// 저장된 필드 인덱스를 사용하여 파라미터 순서대로 추가
		for _, fieldIndex := range fieldIndices {
			if fieldIndex >= v.NumField() {
				return fmt.Errorf("InsertList: %d번째 항목에서 필드 인덱스 %d 접근 오류", idx, fieldIndex)
			}
			fieldValue := v.Field(fieldIndex)
			allParams = append(allParams, fieldValue.Interface())
		}
	}

	// 최종 쿼리 생성
	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES %s",
		tableName,
		strings.Join(columns, ", "),      // 컬럼 목록
		strings.Join(valueStrings, ", "), // 플레이스홀더 목록
	)

	// 로깅 및 쿼리 실행 (전달받은 Tx 객체 사용)
	logSQL(query, allParams)
	result, err := tx.Exec(query, allParams...)
	if err != nil {
		// 쿼리 실패 시 로그에 쿼리와 파라미터 포함
		loghandle.Error("InsertList: 벌크 INSERT 실패 (테이블: %s). Query: %s, Params: %v", tableName, query, allParams)
		return fmt.Errorf("InsertList: 벌크 INSERT 실패 (테이블: %s): %w", tableName, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		loghandle.Warn("InsertList: 영향받은 행 수 확인 실패 (테이블: %s): %v", tableName, err)
	} else {
		loghandle.Debug("InsertList: 테이블 '%s'에 %d개 항목 삽입 성공 (영향받은 행: %d)", tableName, len(items), rowsAffected)
	}

	return nil
}
