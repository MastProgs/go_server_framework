package database

import (
	"context"
	"errors"
	"fmt"
	"go_server_framework/loghandle"
	"strings"
	"time"
)

// User는 사용자 모델을 정의합니다
type User struct {
	BaseModel
	Username  string    `db:"username"`
	Email     string    `db:"email"`
	Password  string    `db:"password"`
	FullName  string    `db:"full_name"`
	IsActive  bool      `db:"is_active"`
	LastLogin time.Time `db:"last_login"`
}

// TableName은 테이블 이름을 반환합니다
func (u *User) TableName() string {
	return "users"
}

// CreateUserTable은 사용자 테이블을 생성합니다
func CreateUserTable() error {
	handler, err := NewHandler()
	if err != nil {
		return fmt.Errorf("데이터베이스 핸들러 생성 실패: %w", err)
	}

	if !handler.IsConnected() {
		return errors.New("데이터베이스 연결이 없습니다")
	}

	query := `
	CREATE TABLE IF NOT EXISTS users (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(50) NOT NULL UNIQUE,
		email VARCHAR(100) NOT NULL UNIQUE,
		password VARCHAR(255) NOT NULL,
		full_name VARCHAR(100) NOT NULL,
		is_active BOOLEAN DEFAULT TRUE,
		last_login DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	_, err = handler.Exec(query)
	if err != nil {
		return fmt.Errorf("사용자 테이블 생성 실패: %w", err)
	}

	return nil
}

// UserRepository는 사용자 모델에 대한 리포지토리입니다
type UserRepository struct {
	repo *Repository
}

// NewUserRepository는 새로운 사용자 리포지토리를 생성합니다
func NewUserRepository() (*UserRepository, error) {
	repo, err := NewRepository()
	if err != nil {
		return nil, err
	}
	return &UserRepository{repo: repo}, nil
}

// IsConnected는 사용자 리포지토리가 유효한 데이터베이스 연결을 가지고 있는지 확인합니다
func (r *UserRepository) IsConnected() bool {
	return r.repo != nil && r.repo.IsConnected()
}

// Create는 새로운 사용자를 생성합니다
func (r *UserRepository) Create(user *User) (int64, error) {
	// 생성 전 훅 호출
	user.BeforeCreate()
	return r.repo.Create(user)
}

// FindByID는 ID로 사용자를 조회합니다
func (r *UserRepository) FindByID(id int64) (*User, error) {
	user := &User{}
	if err := r.repo.FindByID(user, id); err != nil {
		return nil, err
	}
	return user, nil
}

// FindByUsername은 사용자 이름으로 사용자를 조회합니다
func (r *UserRepository) FindByUsername(username string) (*User, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE username = ?", (&User{}).TableName())

	user := &User{}
	row := r.repo.handler.QueryRow(query, username)

	if err := scanModel(row, user); err != nil {
		return nil, fmt.Errorf("사용자 이름으로 사용자 조회 실패: %w", err)
	}

	return user, nil
}

// FindAll은 모든 사용자를 조회합니다
func (r *UserRepository) FindAll() ([]*User, error) {
	var users []*User
	if err := r.repo.FindAll(&User{}, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// Update는 사용자를 업데이트합니다
func (r *UserRepository) Update(user *User) error {
	// 업데이트 전 훅 호출
	user.BeforeUpdate()
	_, err := r.repo.Update(user)
	return err
}

// Delete는 사용자를 삭제합니다
func (r *UserRepository) Delete(user *User) error {
	_, err := r.repo.Delete(user)
	return err
}

// DeleteByID는 ID로 사용자를 삭제합니다
func (r *UserRepository) DeleteByID(id int64) error {
	_, err := r.repo.DeleteByID(&User{}, id)
	return err
}

// Count는 사용자 수를 반환합니다
func (r *UserRepository) Count() (int64, error) {
	return r.repo.Count(&User{})
}

// UpdateLastLogin은 사용자의 마지막 로그인 시간을 업데이트합니다
func (r *UserRepository) UpdateLastLogin(id int64) error {
	query := fmt.Sprintf("UPDATE %s SET last_login = ?, updated_at = ? WHERE id = ?", (&User{}).TableName())

	now := time.Now()
	_, err := r.repo.Exec(query, now, now, id)
	if err != nil {
		return fmt.Errorf("마지막 로그인 시간 업데이트 실패: %w", err)
	}

	return nil
}

// FindActiveUsers는 활성 사용자를 조회합니다
func (r *UserRepository) FindActiveUsers() ([]*User, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE is_active = ?", (&User{}).TableName())

	rows, err := r.repo.Query(query, true)
	if err != nil {
		return nil, fmt.Errorf("활성 사용자 조회 실패: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		if err := scanModel(rows, user); err != nil {
			return nil, fmt.Errorf("사용자 스캔 실패: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("행 반복 중 오류 발생: %w", err)
	}

	return users, nil
}

// CreateUserWithTransaction은 트랜잭션 내에서 사용자를 생성합니다
func (r *UserRepository) CreateUserWithTransaction(user *User) (int64, error) {
	var userID int64

	err := ExecuteWithTransaction(func(tx *Transaction) error {
		// 생성 전 훅 호출
		user.BeforeCreate()

		// 필드 추출
		fields, values, placeholders := extractFieldsAndValues(user)
		if len(fields) == 0 {
			return errors.New("모델에 필드가 없습니다")
		}

		// 쿼리 생성
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			user.TableName(),
			strings.Join(fields, ", "),
			strings.Join(placeholders, ", "))

		// 쿼리 실행
		result, err := tx.Exec(query, values...)
		if err != nil {
			return fmt.Errorf("트랜잭션 내에서 사용자 생성 실패: %w", err)
		}

		// ID 가져오기
		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("마지막 삽입 ID 가져오기 실패: %w", err)
		}

		userID = id
		return nil
	})

	if err != nil {
		return 0, err
	}

	return userID, nil
}

// 사용 예제
func UserRepositoryExample() {
	// 사용자 테이블 생성
	if err := CreateUserTable(); err != nil {
		loghandle.Error("사용자 테이블 생성 실패: %v", err)
		loghandle.Warn("데이터베이스 연결 없이 계속합니다")
	} else {
		loghandle.Info("사용자 테이블 생성 성공")
	}

	// 사용자 리포지토리 생성
	userRepo, err := NewUserRepository()
	if err != nil {
		loghandle.Error("사용자 리포지토리 생성 실패: %v", err)
		return
	}

	// 데이터베이스 연결 확인
	if !userRepo.IsConnected() {
		loghandle.Warn("데이터베이스 연결이 없어 사용자 예제를 실행할 수 없습니다")
		return
	}

	// 새 사용자 생성
	user := &User{
		Username: "johndoe",
		Email:    "john.doe@example.com",
		Password: "hashed_password",
		FullName: "John Doe",
		IsActive: true,
	}

	userID, err := userRepo.Create(user)
	if err != nil {
		loghandle.Error("사용자 생성 실패: %v", err)
		return
	}

	loghandle.Info("사용자 생성 성공: ID = %d", userID)

	// ID로 사용자 조회
	foundUser, err := userRepo.FindByID(userID)
	if err != nil {
		loghandle.Error("사용자 조회 실패: %v", err)
		return
	}

	loghandle.Info("사용자 조회 성공: %s (%s)", foundUser.Username, foundUser.Email)

	// 사용자 업데이트
	foundUser.FullName = "John Smith Doe"
	if err := userRepo.Update(foundUser); err != nil {
		loghandle.Error("사용자 업데이트 실패: %v", err)
		return
	}

	loghandle.Info("사용자 업데이트 성공")

	// 컨텍스트를 사용한 예제
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 컨텍스트를 사용하여 모든 사용자 조회
	var users []*User
	if err := userRepo.repo.FindAllContext(ctx, &User{}, &users); err != nil {
		loghandle.Error("모든 사용자 조회 실패: %v", err)
		return
	}

	loghandle.Info("총 사용자 수: %d", len(users))

	// 트랜잭션을 사용한 예제
	newUser := &User{
		Username: "janedoe",
		Email:    "jane.doe@example.com",
		Password: "hashed_password",
		FullName: "Jane Doe",
		IsActive: true,
	}

	newUserID, err := userRepo.CreateUserWithTransaction(newUser)
	if err != nil {
		loghandle.Error("트랜잭션을 사용한 사용자 생성 실패: %v", err)
		return
	}

	loghandle.Info("트랜잭션을 사용한 사용자 생성 성공: ID = %d", newUserID)
}
