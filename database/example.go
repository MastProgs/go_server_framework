package database

import (
	"fmt"
	"strings"
	"time"

	"go_server_framework/loghandle"
)

// TblTest는 예제 테이블 모델입니다
type TblTest struct {
	ID        int64     `pk:"true" auto:"true"`       // db 태그 없이 자동으로 "id"로 변환
	Name      string    `length:"10"`                 // db 태그 없이 자동으로 "name"으로 변환
	Value     float64   `default:"0.0"`               // db 태그 없이 자동으로 "value"로 변환
	Active    bool      `default:"1"`                 // db 태그 없이 자동으로 "active"로 변환
	CreatedAt time.Time `default:"CURRENT_TIMESTAMP"` // db 태그 없이 자동으로 "created_at"으로 변환
	UpdatedAt time.Time `default:"CURRENT_TIMESTAMP"` // db 태그 없이 자동으로 "updated_at"으로 변환
	UserEmail string    `Null:"true" length:"10"`
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
	err = repo.FindOneByQuery(&result, "SELECT * FROM tbl_test WHERE id = ?", id)
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

	// UPSERT 예제
	upsertRecord := TblTest{
		ID:     id,
		Name:   "UPSERT 테스트",
		Value:  555.55,
		Active: true,
	}

	upsertID, err := repo.UpsertStruct(upsertRecord)
	if err != nil {
		loghandle.Error("UPSERT 오류: %v", err)
		return
	}
	loghandle.Info("UPSERT 결과: %d", upsertID)

	// 다수 레코드 조회
	var records []TblTest
	err = repo.FindAllByQuery(&records, "SELECT * FROM tbl_test WHERE active = ?", true)
	if err != nil {
		loghandle.Error("다수 데이터 조회 오류: %v", err)
		return
	}
	loghandle.Info("조회된 레코드 수: %d", len(records))

	// SELECT ORM 사용 예제 ------------------------------
	// 1. 단일 레코드 조회 예제
	loghandle.Info("=== 단일 레코드 조회 예제 ===")
	user := TblTest{}
	err = repo.FindOne(&user, map[string]interface{}{"id": id})
	if err != nil {
		loghandle.Error("단일 레코드 조회 오류: %v", err)
	} else {
		loghandle.Info("조회된 사용자: %+v", user)
	}

	// 2. 특정 컬럼만 선택하여 조회
	loghandle.Info("=== 특정 컬럼만 선택하여 조회 예제 ===")
	selectedUser := TblTest{}
	err = repo.Find(&selectedUser, &FindOptions{
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
	err = repo.FindAll(&users, &FindOptions{
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
	err = repo.Find(&inUsers, &FindOptions{
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
	var nullUsers []TblTest
	err = repo.Find(&nullUsers, &FindOptions{
		Where: map[string]interface{}{
			"user_email__null": false, // NOT NULL 조건
		},
	})
	if err != nil {
		loghandle.Error("NULL 처리 오류: %v", err)
	} else {
		loghandle.Info("이메일이 있는 사용자 수: %d", len(nullUsers))
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
	err = repo.Find(&pagedUsers, options)
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
	if err := batch.AddDelete(deleteRecord, map[string]interface{}{"id": 2}); err != nil {
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
