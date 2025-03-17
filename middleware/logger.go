package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ANSI 색상 코드
const (
	Reset      = "\033[0m"
	Red        = "\033[31m"
	Green      = "\033[32m"
	Yellow     = "\033[33m"
	Blue       = "\033[34m"
	Purple     = "\033[35m"
	Cyan       = "\033[36m"
	White      = "\033[37m"
	BoldRed    = "\033[1;31m"
	BoldGreen  = "\033[1;32m"
	BoldYellow = "\033[1;33m"
	BoldBlue   = "\033[1;34m"
)

// 메서드별 색상
var methodColors = map[string]string{
	"GET":     Green,
	"POST":    Blue,
	"PUT":     Yellow,
	"DELETE":  Red,
	"PATCH":   Purple,
	"HEAD":    Cyan,
	"OPTIONS": White,
}

// statusCodeColor는 상태 코드에 따라 색상을 반환합니다
func statusCodeColor(code int) string {
	switch {
	case code >= 200 && code < 300:
		return Green
	case code >= 300 && code < 400:
		return Cyan
	case code >= 400 && code < 500:
		return Yellow
	case code >= 500:
		return Red
	default:
		return Reset
	}
}

// ResponseWriter 래퍼
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int
}

// WriteHeader는 상태 코드를 저장합니다
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write는 쓰여진 바이트 수를 추적합니다
func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += n
	return n, err
}

// 컨텍스트 키 타입
type contextKey string

// 컨텍스트 키 상수
const (
	originalPathKey  contextKey = "originalPath"
	requestLoggedKey contextKey = "requestLogged"
)

// 이미 로깅된 요청을 추적하기 위한 맵과 뮤텍스
var (
	loggedRequests   = make(map[string]time.Time)
	loggedRequestsMu sync.Mutex
)

// RequestLoggerMiddleware는 HTTP 요청 정보를 콘솔에 출력하는 미들웨어입니다
func RequestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 요청 시작 시간
		startTime := time.Now()

		// 원본 요청 경로 확인
		originalPath := r.URL.Path

		// 컨텍스트에서 원본 경로 확인 (이미 설정되어 있을 수 있음)
		if path, ok := r.Context().Value(originalPathKey).(string); ok && path != "" {
			originalPath = path
		}

		// 이미 로깅된 요청인지 확인
		if logged, ok := r.Context().Value(requestLoggedKey).(bool); ok && logged {
			// 이미 로깅된 요청이면 로깅하지 않고 다음 핸들러로 진행
			next.ServeHTTP(w, r)
			return
		}

		// 클라이언트 IP 가져오기
		clientIP := r.RemoteAddr
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			clientIP = forwardedFor
		}

		// 요청 ID 생성 (IP + 메서드 + 원본 경로)
		requestID := fmt.Sprintf("%s-%s-%s", clientIP, r.Method, originalPath)

		// 중복 로깅 방지를 위한 검사
		shouldLog := true

		loggedRequestsMu.Lock()

		// 이미 로깅된 요청인지 확인
		if lastTime, exists := loggedRequests[requestID]; exists {
			// 같은 요청이 최근 10ms 이내에 로깅되었다면 로깅하지 않음
			if time.Since(lastTime) < 10*time.Millisecond {
				shouldLog = false
			}
		}

		// 중첩 경로 검사 (예: /protected/data가 /data를 포함하는 경우)
		if shouldLog && strings.Count(originalPath, "/") > 1 {
			// 마지막 세그먼트만 추출
			lastSegment := originalPath
			if idx := strings.LastIndex(originalPath, "/"); idx >= 0 {
				lastSegment = originalPath[idx:]
			}

			// 다른 경로에서 같은 마지막 세그먼트를 가진 요청이 있는지 확인
			for path, t := range loggedRequests {
				// 최근 10ms 이내의 요청만 검사
				if time.Since(t) < 10*time.Millisecond {
					// 같은 마지막 세그먼트를 가진 다른 경로가 있으면 로깅하지 않음
					if strings.HasSuffix(path, "-"+r.Method+"-"+lastSegment) && path != requestID {
						shouldLog = false
						break
					}
				}
			}
		}

		// 현재 요청을 로깅된 요청 맵에 추가
		if shouldLog {
			loggedRequests[requestID] = startTime
		}

		// 맵 크기 제한 (메모리 누수 방지)
		if len(loggedRequests) > 1000 {
			// 오래된 항목 제거 (1초 이상 지난 항목)
			now := time.Now()
			for id, t := range loggedRequests {
				if now.Sub(t) > 1*time.Second {
					delete(loggedRequests, id)
				}
			}

			// 여전히 크기가 크면 맵 초기화
			if len(loggedRequests) > 900 {
				loggedRequests = make(map[string]time.Time)
			}
		}

		loggedRequestsMu.Unlock()

		// 원본 경로와 로깅 상태를 컨텍스트에 저장
		ctx := context.WithValue(r.Context(), originalPathKey, originalPath)
		ctx = context.WithValue(ctx, requestLoggedKey, true)
		r = r.WithContext(ctx)

		// 응답 래퍼 생성
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // 기본값
		}

		// 다음 핸들러 호출
		next.ServeHTTP(rw, r)

		// 로깅 여부 결정
		if shouldLog {
			// 요청 처리 시간 계산
			duration := time.Since(startTime)

			// 요청 헤더 정보
			userAgent := r.Header.Get("User-Agent")
			if userAgent == "" {
				userAgent = "-"
			}

			// 메서드 색상
			methodColor, ok := methodColors[r.Method]
			if !ok {
				methodColor = Reset
			}

			// 상태 코드 색상
			statusColor := statusCodeColor(rw.statusCode)

			// 한 줄로 요청 정보 출력
			fmt.Printf("%s[%s]%s %s%d%s %s%s%s %s%.2fms%s %s%s%s %s%s%s %s%s%s %s%d bytes%s\n",
				Yellow, startTime.Format("2006-01-02 15:04:05.000"), Reset,
				statusColor, rw.statusCode, Reset,
				methodColor, r.Method, Reset,
				BoldBlue, float64(duration.Microseconds())/1000.0, Reset,
				Cyan, clientIP, Reset,
				Blue, originalPath, Reset, // 원본 경로 사용
				Purple, truncateString(userAgent, 30), Reset,
				Green, rw.written, Reset,
			)
		}
	})
}

// truncateString은 문자열을 지정된 길이로 자릅니다
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
