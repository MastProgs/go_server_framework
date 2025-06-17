package database

import (
	"fmt"
	"strings"
	"time"

	"go_server_framework/loghandle"
)

// TblTest는 예제 테이블 모델입니다 - 현재 Null:"true" 는 정상작동 하지 않음
// 테이블 타입이 TblTest 인 경우, 테이블 이름은 "tbl_test" 로 자동 변환됩니다.
// Id 필드 또한 네이밍을 ID 로 하게 되는경우, 대문자를 기준으로 자동 변환하여 "i_d" 로 변환되기 때문에, 네이밍 규칙을 지키는것이 좋습니다.
// CreatedAt 필드 또한, 대문자를 기준으로 자동 변환하여 "created_at" 으로 변환됩니다.
type TblTest struct {
	Id        int64     `pk:"true" auto:"true"`                                   // db 태그 없이 자동으로 "id"로 변환
	Name      string    `default:"" length:"100"`                                 // db 태그 없이 자동으로 "name"으로 변환
	Value     float64   `default:"0.0"`                                           // db 태그 없이 자동으로 "value"로 변환
	Active    bool      `default:"1"`                                             // db 태그 없이 자동으로 "active"로 변환
	CreatedAt time.Time `default:"CURRENT_TIMESTAMP"`                             // db 태그 없이 자동으로 "created_at"으로 변환
	UpdatedAt time.Time `default:"CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"` // 자동으로 업데이트됨
	UserEmail string    `default:"" length:"100"`
}

// ExampleUsage는 ORM 사용 예제를 보여줍니다
func ExampleUsage() {
	// 로그 활성화
	EnableSQLLogging = true

	// 리포지토리 생성
	repo, err := NewRepository()
	if err != nil {
		loghandle.Error("리포지토리 생성 오류: %v", err)
		return
	}

	// 테이블 존재 여부 확인
	testModel := TblTest{}
	exists, err := repo.IsTableExists(getTableName(testModel))
	if err != nil {
		loghandle.Error("테이블 존재 여부 확인 오류: %v", err)
		return
	}

	if exists {
		loghandle.Info("테이블 '%s'가 이미 존재합니다", getTableName(testModel))

		// 기존 테이블 재생성 필요시 (스키마 변경 등)
		// 첫 번째 매개변수: ifNotExists (false로 설정, 테이블 삭제할 것이므로 의미 없음)
		// 두 번째 매개변수: dropIfExists (true로 설정, 테이블을 삭제하고 다시 생성)
		// 세 번째 매개변수: backupBeforeDrop (true로 설정, 삭제 전 백업)
		err = repo.CreateTableFromStruct(testModel, false, true, true)
		if err != nil {
			loghandle.Error("테이블 재생성 오류: %v", err)
			return
		}
	} else {
		loghandle.Info("테이블 '%s'가 존재하지 않습니다. 새로 생성합니다.", getTableName(testModel))

		// 테이블 생성 (이미 존재하면 생성하지 않음)
		err = repo.CreateTableFromStruct(testModel, true)
		if err != nil {
			loghandle.Error("테이블 생성 오류: %v", err)
			return
		}
	}

	// INSERT 예제
	newRecord := TblTest{
		Name:   "테스트 데이터",
		Value:  123.45,
		Active: true,
	}

	id, err := repo.InsertStruct(newRecord)
	if err != nil {
		loghandle.Error("데이터 삽입 오류: %v", err)
		return
	}
	loghandle.Info("삽입된 ID: %d", id)

	// 단일 레코드 조회
	var result TblTest
	_, err = repo.FindOneByQuery(&result, "SELECT * FROM tbl_test WHERE id = ?", id)
	if err != nil {
		loghandle.Error("데이터 조회 오류: %v", err)
		return
	}
	loghandle.Info("조회 결과: %+v", result)

	// UPDATE 예제
	result.Name = "수정된 이름"
	result.Value = 987.65
	result.UpdatedAt = time.Now()

	affected, err := repo.UpdateStruct(result, map[string]interface{}{"id": id})
	if err != nil {
		loghandle.Error("데이터 업데이트 오류: %v", err)
		return
	}
	loghandle.Info("업데이트된 행 수: %d", affected)

	// UpdateFields 예제 (특정 필드만 업데이트)
	loghandle.Info("=== UpdateFields 예제 ===")
	fieldsToUpdate := map[string]interface{}{
		"name":  "선택적으로 업데이트된 이름",
		"value": 123.456,
	}

	affected, err = repo.UpdateFields(getTableName(result), fieldsToUpdate, map[string]interface{}{"id": id})
	if err != nil {
		loghandle.Error("선택적 필드 업데이트 오류: %v", err)
	} else {
		loghandle.Info("선택적 필드 업데이트 행 수: %d", affected)
	}

	// UpdateFieldsByStruct 예제
	loghandle.Info("=== UpdateFieldsByStruct 예제 ===")
	updatedRecord := TblTest{
		Name:      "구조체로 업데이트된 이름",
		Value:     789.012,
		Active:    false,
		UserEmail: "test@example.com",
	}

	// 업데이트할 필드 이름 목록
	fieldNames := []string{"Name", "Value"}

	affected, err = repo.UpdateFieldsByStruct(updatedRecord, fieldNames, map[string]interface{}{"id": id})
	if err != nil {
		loghandle.Error("구조체 선택적 필드 업데이트 오류: %v", err)
	} else {
		loghandle.Info("구조체 선택적 필드 업데이트 행 수: %d", affected)
	}

	// UpdateNonZero 예제 - zero value가 아닌 필드만 업데이트
	loghandle.Info("=== UpdateNonZero 예제 ===")
	nonZeroRecord := TblTest{
		// Id 필드는 기본 키이므로 무시됨
		Name:   "Zero가 아닌 필드만 업데이트", // 이 필드만 업데이트됨
		Value:  0,                   // zero value이므로 업데이트되지 않음
		Active: false,               // bool의 경우 false가 zero value이므로 업데이트되지 않음
	}

	affected, err = repo.UpdateNonZero(nonZeroRecord, map[string]interface{}{"id": id})
	if err != nil {
		loghandle.Error("Zero가 아닌 필드 업데이트 오류: %v", err)
	} else {
		loghandle.Info("Zero가 아닌 필드 업데이트 행 수: %d", affected)
	}

	// 원본과 비교하여 변경된 필드만 업데이트
	loghandle.Info("=== 원본과 비교하여 변경된 필드만 업데이트 ===")
	// 원본 데이터 조회
	var originalRecord TblTest
	_, err = repo.FindOne(&originalRecord, map[string]interface{}{"id": id})
	if err != nil {
		loghandle.Error("원본 데이터 조회 오류: %v", err)
	} else {
		// 원본과 변경된 데이터 준비
		modifiedRecord := originalRecord     // 원본 복사
		modifiedRecord.Name = "변경된 이름만 업데이트" // 이 필드만 변경

		// 원본과 비교하여 변경된 필드만 업데이트
		affected, err = repo.UpdateNonZero(modifiedRecord, map[string]interface{}{"id": id}, originalRecord)
		if err != nil {
			loghandle.Error("변경된 필드 업데이트 오류: %v", err)
		} else {
			loghandle.Info("변경된 필드 업데이트 행 수: %d", affected)
		}
	}

	// UPSERT 예제
	upsertRecord := TblTest{
		Id:        id,
		Name:      "UPSERT 테스트",
		Value:     555.55,
		Active:    true,
		CreatedAt: time.Now(),
	}

	upsertID, err := repo.UpsertStruct(upsertRecord)
	if err != nil {
		loghandle.Error("UPSERT 오류: %v", err)
		return
	}
	loghandle.Info("UPSERT 결과: %d", upsertID)

	// UpsertNonZero 예제 - zero가 아닌 필드만 업데이트
	loghandle.Info("=== UpsertNonZero 예제 ===")
	nonZeroUpsertRecord := TblTest{
		Id:   id, // 기존 레코드 업데이트를 위한 ID
		Name: "NonZero로 업데이트된 이름",
		// Value와 Active 필드는 zero 값이므로 업데이트되지 않음
	}

	nonZeroID, err := repo.UpsertNonZero(nonZeroUpsertRecord)
	if err != nil {
		loghandle.Error("UpsertNonZero 오류: %v", err)
	} else {
		loghandle.Info("UpsertNonZero 결과: %d", nonZeroID)

		// 결과 확인
		var updatedRecord TblTest
		_, err = repo.FindOne(&updatedRecord, map[string]interface{}{"id": nonZeroID})
		if err != nil {
			loghandle.Error("업데이트된 레코드 조회 오류: %v", err)
		} else {
			loghandle.Info("UpsertNonZero 결과 레코드: %+v", updatedRecord)
		}
	}

	// UPSERT 예제
	upsertNewRecord := TblTest{
		Id:        9999,
		Name:      "새로운 UPSERT 삽입 테스트",
		Value:     555.55,
		Active:    true,
		CreatedAt: time.Now(),
	}

	upsertNewID, err := repo.UpsertStruct(upsertNewRecord)
	if err != nil {
		loghandle.Error("UPSERT 오류: %v", err)
		return
	}
	loghandle.Info("UPSERT 결과: %d", upsertNewID)

	// 다수 레코드 조회
	var records []TblTest
	_, err = repo.FindAll(&records, &FindOptions{
		Where: map[string]interface{}{"active": true},
	})
	if err != nil {
		loghandle.Error("다수 데이터 조회 오류: %v", err)
		return
	}
	loghandle.Info("조회된 레코드 수: %d", len(records))

	// SELECT ORM 사용 예제 ------------------------------
	// 1. 단일 레코드 조회 예제
	loghandle.Info("=== 단일 레코드 조회 예제 ===")
	user := TblTest{}
	_, err = repo.FindOne(&user, map[string]interface{}{"id": id})
	if err != nil {
		loghandle.Error("단일 레코드 조회 오류: %v", err)
	} else {
		loghandle.Info("조회된 사용자: %+v", user)
	}

	// 2. 특정 컬럼만 선택하여 조회
	loghandle.Info("=== 특정 컬럼만 선택하여 조회 예제 ===")
	selectedUser := TblTest{}
	_, err = repo.Find(&selectedUser, &FindOptions{
		Where:   map[string]interface{}{"id": id},
		Columns: []string{"id", "name"},
	})
	if err != nil {
		loghandle.Error("특정 컬럼 조회 오류: %v", err)
	} else {
		loghandle.Info("선택적 컬럼 조회 결과: %+v", selectedUser)
	}

	// 3. 다양한 조건으로 여러 레코드 조회
	loghandle.Info("=== 다양한 조건으로 여러 레코드 조회 예제 ===")
	var users []TblTest
	_, err = repo.FindAll(&users, &FindOptions{
		Where: map[string]interface{}{
			"value__gt":  50.0, // value > 50.0
			"active":     true,
			"name__like": "%테스트%",
		},
		OrderBy: []string{"value DESC", "id"},
		Limit:   10,
		Offset:  0,
	})
	if err != nil {
		loghandle.Error("다양한 조건 조회 오류: %v", err)
	} else {
		loghandle.Info("조회된 레코드 수: %d", len(users))
		for i, u := range users {
			loghandle.Info("  %d: %+v", i+1, u)
		}
	}

	// 4. IN 조건 사용 예제
	loghandle.Info("=== IN 조건 사용 예제 ===")
	var inUsers []TblTest
	_, err = repo.Find(&inUsers, &FindOptions{
		Where: map[string]interface{}{
			"id__in": []int64{1, 2, 3, 5, 8},
		},
	})
	if err != nil {
		loghandle.Error("IN 조건 조회 오류: %v", err)
	} else {
		loghandle.Info("IN 조건으로 조회된 레코드 수: %d", len(inUsers))
	}

	// 5. GROUP BY 및 COUNT 예제
	loghandle.Info("=== 레코드 카운트 예제 ===")
	count, err := repo.Count(&TblTest{}, map[string]interface{}{"active": true})
	if err != nil {
		loghandle.Error("카운트 오류: %v", err)
	} else {
		loghandle.Info("활성 사용자 수: %d", count)
	}

	// 6. NULL 값 처리 예제
	loghandle.Info("=== NULL 값 처리 예제 ===")

	// NULL 값이 있는 테스트 데이터 삽입
	nullTest := &TblTest{
		Name:   "NULL 테스트",
		Value:  123,
		Active: true,
	}
	// UserEmail 필드는 의도적으로 NULL로 둠

	nullID, err := repo.InsertStruct(*nullTest)
	if err != nil {
		loghandle.Error("NULL 테스트 데이터 삽입 오류: %v", err)
	} else {
		loghandle.Info("NULL 테스트 데이터 ID: %d", nullID)
	}

	// NULL 값이 있는 레코드 조회
	var nullRecord TblTest
	_, err = repo.FindOne(&nullRecord, map[string]interface{}{
		"id": nullID,
	})

	if err != nil {
		loghandle.Error("NULL 값 레코드 조회 오류: %v", err)
	} else {
		loghandle.Info("NULL 값 레코드: %+v", nullRecord)
		loghandle.Info("  - Email 필드 (NULL): '%s'", nullRecord.UserEmail)
	}

	// NULL 조건으로 검색
	var nullUsers []TblTest
	_, err = repo.Find(&nullUsers, &FindOptions{
		Where: map[string]interface{}{
			"user_email__null": true, // user_email이 NULL인 레코드 조회
		},
	})

	if err != nil {
		loghandle.Error("NULL 처리 오류: %v", err)
	} else {
		loghandle.Info("이메일이 NULL인 사용자 수: %d", len(nullUsers))
		for i, u := range nullUsers {
			loghandle.Info("  %d: %+v", i+1, u)
		}
	}

	// NULL이 아닌 조건으로 검색
	var notNullUsers []TblTest
	_, err = repo.Find(&notNullUsers, &FindOptions{
		Where: map[string]interface{}{
			"user_email__null": false, // user_email이 NULL이 아닌 레코드 조회
		},
	})

	if err != nil {
		loghandle.Error("NULL 아님 처리 오류: %v", err)
	} else {
		loghandle.Info("이메일이 NULL이 아닌 사용자 수: %d", len(notNullUsers))
	}

	// 7. 완전히 동적인 쿼리 구성 예제
	loghandle.Info("=== 동적 쿼리 구성 예제 ===")
	options := &FindOptions{}

	// 검색어가 있으면 추가
	searchTerm := "테스트"
	if searchTerm != "" {
		options.Where = map[string]interface{}{
			"name__like": "%" + searchTerm + "%",
		}
	}

	// 정렬 설정
	sortField := "value"
	sortOrder := "DESC"
	if sortField != "" {
		if sortOrder == "DESC" {
			options.OrderBy = []string{sortField + " DESC"}
		} else {
			options.OrderBy = []string{sortField}
		}
	}

	// 페이지네이션
	page := 1
	pageSize := 5
	options.Limit = pageSize
	options.Offset = (page - 1) * pageSize

	var pagedUsers []TblTest
	_, err = repo.Find(&pagedUsers, options)
	if err != nil {
		loghandle.Error("동적 쿼리 오류: %v", err)
	} else {
		loghandle.Info("페이지 %d의 조회 결과: %d개 레코드", page, len(pagedUsers))
	}

	// ---------------------------------------------------

	// 배치 처리 예제
	BatchExample()
}

// BatchExample은 배치 처리 사용 예제를 보여줍니다
func BatchExample() {
	// 리포지토리 생성
	repo, err := NewRepository()
	if err != nil {
		loghandle.Error("리포지토리 생성 오류: %v", err)
		return
	}

	// 배치 작업 생성
	batch, err := repo.NewBatch()
	if err != nil {
		loghandle.Error("배치 작업 생성 오류: %v", err)
		return
	}

	// 다수의 INSERT 작업 추가
	for i := 1; i <= 5; i++ {
		record := TblTest{
			Name:   fmt.Sprintf("배치 테스트 %d", i),
			Value:  float64(i * 10),
			Active: true,
		}

		if err := batch.AddInsert(record); err != nil {
			loghandle.Error("배치 INSERT 추가 오류: %v", err)
			batch.Rollback()
			return
		}
	}

	// UPDATE 작업 추가
	updateRecord := TblTest{
		Name:   "업데이트된 배치 레코드",
		Value:  999.99,
		Active: false,
	}

	if err := batch.AddUpdate(updateRecord, map[string]interface{}{"id": 1}); err != nil {
		loghandle.Error("배치 UPDATE 추가 오류: %v", err)
		batch.Rollback()
		return
	}

	// DELETE 작업 추가
	deleteRecord := TblTest{}
	if err := batch.AddDelete(deleteRecord, map[string]interface{}{"id": 10000}); err != nil {
		loghandle.Error("배치 DELETE 추가 오류: %v", err)
		batch.Rollback()
		return
	}

	// 배치 실행
	affected, err := batch.Execute()
	if err != nil {
		loghandle.Error("배치 실행 오류: %v", err)
		return
	}

	loghandle.Info("배치 작업으로 영향받은 행 수: %d", affected)
}

// TestFieldNamingExample은 필드 이름 자동 변환 기능을 보여줍니다
func TestFieldNamingExample() {
	EnableSQLLogging = true

	// 필드 이름 자동 변환 테스트
	loghandle.Info("=== 필드 이름 자동 변환 테스트 ===")
	loghandle.Info("UserComplexEntity 필드 변환:")
	loghandle.Info("UserID -> user_id")
	loghandle.Info("FirstName -> first_name")
	loghandle.Info("LastName -> last_name")
	loghandle.Info("EmailAddress -> email_address")
	loghandle.Info("PasswordHash -> password_hash")
	loghandle.Info("IsAdmin -> is_admin")
	loghandle.Info("LoginCount -> login_count")
	loghandle.Info("LastLoginTime -> last_login_time")
	loghandle.Info("RegistrationDate -> registration_date")

	// 테이블 이름 자동 변환 테스트
	loghandle.Info("=== 테이블 이름 자동 변환 테스트 ===")
	loghandle.Info("ProductItem -> product_item")
	loghandle.Info("OrderDetail -> order_detail")
	loghandle.Info("UserComplexEntity -> user_complex_entity")

	// 예제 데이터베이스 쿼리 실행
	type UserComplexEntity struct {
		UserID           int64     `pk:"true" auto:"true"`
		FirstName        string    `length:"50"`
		LastName         string    `length:"50"`
		EmailAddress     string    `length:"100"`
		PasswordHash     string    `length:"255"`
		IsAdmin          bool      `default:"0"`
		LoginCount       int       `default:"0"`
		LastLoginTime    time.Time `Null:"true"`
		RegistrationDate time.Time `default:"CURRENT_TIMESTAMP"`
	}

	// ProductItem에 대한 테이블 이름
	productTableName := "products"

	// 레코드 삽입 예제
	newUser := UserComplexEntity{
		FirstName:    "John",
		LastName:     "Doe",
		EmailAddress: "john.doe@example.com",
		PasswordHash: "hashed_password",
		IsAdmin:      true,
	}

	// INSERT 쿼리 준비
	columns := []string{
		"first_name",
		"last_name",
		"email_address",
		"password_hash",
		"is_admin",
	}
	values := []interface{}{
		newUser.FirstName,
		newUser.LastName,
		newUser.EmailAddress,
		newUser.PasswordHash,
		newUser.IsAdmin,
	}

	// 수동으로 쿼리 실행
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (?, ?, ?, ?, ?)",
		productTableName,
		strings.Join(columns, ", "),
	)

	if EnableSQLLogging {
		logSQL(query, values)
	}

	// 직접 핸들러 사용
	handler, _ := NewHandler()
	result, err := handler.Exec(query, values...)
	if err != nil {
		loghandle.Error("사용자 삽입 오류: %v", err)
		return
	}

	id, _ := result.LastInsertId()
	loghandle.Info("삽입된 사용자 ID: %d", id)
}

// TransactionExample은 트랜잭션 사용 예제를 보여줍니다
func TransactionExample() {
	// ExecuteWithTransaction 함수를 사용한 트랜잭션 예제
	err := ExecuteWithTransaction(func(tx *Transaction) error {
		// INSERT 쿼리
		result, err := tx.Exec(
			"INSERT INTO tbl_test (name, value, active) VALUES (?, ?, ?)",
			"트랜잭션 테스트", 123.45, true,
		)
		if err != nil {
			return err
		}

		id, err := result.LastInsertId()
		if err != nil {
			return err
		}

		loghandle.Info("트랜잭션 내 삽입된 ID: %d", id)

		// UPDATE 쿼리
		_, err = tx.Exec(
			"UPDATE tbl_test SET value = ? WHERE id = ?",
			999.99, id,
		)
		if err != nil {
			return err
		}

		// 조건부 롤백 예제
		if id > 100 {
			return fmt.Errorf("ID가 너무 큽니다: %d", id)
		}

		return nil
	})

	if err != nil {
		loghandle.Error("트랜잭션 실행 오류: %v", err)
		return
	}

	loghandle.Info("트랜잭션이 성공적으로 완료되었습니다.")
}
