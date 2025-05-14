package middleware

import (
	"net/http"
)

// CORSMiddleware는 CORS(Cross-Origin Resource Sharing) 설정을 처리하는 미들웨어입니다.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS 헤더 설정
		w.Header().Set("Access-Control-Allow-Origin", "*") // 모든 도메인 허용 또는 특정 도메인 지정 가능
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// OPTIONS 메서드 요청 처리 (프리플라이트 요청)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent) // 204 상태 코드 반환
			return
		}

		// 다음 핸들러로 진행
		next.ServeHTTP(w, r)
	})
}
