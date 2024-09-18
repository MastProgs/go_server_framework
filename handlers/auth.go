package handlers

import (
	"go_server_framework/config"
	"go_server_framework/types"

	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var cfg = config.GetConfig()
var jwtSecret = []byte(cfg.JWT.Secret)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	// 여기서 사용자 인증 로직을 구현합니다.
	// 예를 들어, 사용자 이름과 비밀번호를 확인하고 유효한 경우 토큰을 생성합니다.

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 123, // 예시 사용자 ID
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := types.CreateSuccessResponse(map[string]string{"token": tokenString})
	types.SendJSONResponse(w, response)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// JWT는 stateless이므로 서버 측에서 특별한 로그아웃 처리가 필요 없습니다.
	// 클라이언트 측에서 토큰을 삭제하도록 안내하는 메시지를 보낼 수 있습니다.
	response := types.CreateSuccessResponse(map[string]string{"message": "Logged out successfully"})
	types.SendJSONResponse(w, response)
}
