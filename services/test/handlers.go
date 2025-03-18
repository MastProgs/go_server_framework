package test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"

	"go_server_framework/config"
	"go_server_framework/database"
	"go_server_framework/loghandle"
)

// PingHandler는 ping 요청을 처리합니다
func PingHandler(r *http.Request) (interface{}, error) {
	data := map[string]string{"msg": "pong from test service"}
	return data, nil
}

// EchoHandler는 요청 본문을 그대로 반환합니다
func EchoHandler(r *http.Request) (interface{}, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("요청 본문 읽기 실패: %v", err)
	}
	defer r.Body.Close()

	data := map[string]interface{}{
		"msg":  "echo from test service",
		"body": string(body),
	}
	return data, nil
}

func DBTestHandler(r *http.Request) (interface{}, error) {
	database.ExampleUsage()
	return nil, nil
}

// ProtectedHandler는 JWT 인증이 필요한 보호된 엔드포인트입니다
func ProtectedHandler(r *http.Request) (interface{}, error) {
	// 인증된 사용자 정보는 JWT 미들웨어에서 이미 검증되었습니다
	data := map[string]interface{}{
		"message": "이 데이터는 보호되어 있으며 인증된 사용자만 접근할 수 있습니다",
		"time":    time.Now().Format(time.RFC3339),
	}
	return data, nil
}

// LoginHandler는 테스트 로그인 요청을 처리하고 JWT 토큰을 발급합니다
func LoginHandler(r *http.Request) (interface{}, error) {
	// 요청 본문 읽기
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("요청 본문 읽기 실패: %v", err)
	}
	defer r.Body.Close()

	// JSON 파싱
	var loginReq LoginRequest
	if err := json.Unmarshal(body, &loginReq); err != nil {
		return nil, fmt.Errorf("JSON 파싱 실패: %v", err)
	}

	// 간단한 인증 체크 (테스트용)
	if loginReq.Username == "" || loginReq.Password == "" {
		return nil, fmt.Errorf("사용자 이름과 비밀번호를 입력해주세요")
	}

	// 테스트용 인증 (실제로는 데이터베이스 조회 등을 수행)
	if !(loginReq.Username == "admin" && loginReq.Password == "password") &&
		!(loginReq.Username == "user" && loginReq.Password == "password") {
		return nil, fmt.Errorf("잘못된 사용자 이름 또는 비밀번호입니다")
	}

	// JWT 토큰 생성
	accessToken, refreshToken, expiresIn, err := generateTokens(loginReq.Username)
	if err != nil {
		return nil, fmt.Errorf("토큰 생성 실패: %v", err)
	}

	// 응답 생성
	response := TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
	}

	loghandle.Info("로그인 성공 - 사용자: %s", loginReq.Username)
	return response, nil
}

// generateTokens는 액세스 토큰과 리프레시 토큰을 생성합니다
func generateTokens(username string) (string, string, int, error) {
	// JWT 설정 가져오기
	jwtConfig := config.GetConfig().JWT

	// 현재 시간
	now := time.Now()

	// 액세스 토큰 만료 시간
	expirationTime := now.Add(time.Duration(jwtConfig.Expiration) * time.Second)

	// 액세스 토큰 클레임 설정
	accessClaims := jwt.MapClaims{
		"sub":  username,
		"exp":  expirationTime.Unix(),
		"iat":  now.Unix(),
		"role": getRole(username),
	}

	// 액세스 토큰 생성
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(jwtConfig.Secret))
	if err != nil {
		return "", "", 0, err
	}

	// 리프레시 토큰 만료 시간
	refreshExpirationTime := now.Add(time.Duration(jwtConfig.RefreshExpiration) * time.Second)

	// 리프레시 토큰 클레임 설정
	refreshClaims := jwt.MapClaims{
		"sub": username,
		"exp": refreshExpirationTime.Unix(),
		"iat": now.Unix(),
	}

	// 리프레시 토큰 생성
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(jwtConfig.Secret))
	if err != nil {
		return "", "", 0, err
	}

	return accessTokenString, refreshTokenString, jwtConfig.Expiration, nil
}

// getRole은 사용자 이름에 따라 역할을 반환합니다 (테스트용)
func getRole(username string) string {
	if username == "admin" {
		return "admin"
	}
	return "user"
}
