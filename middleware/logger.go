package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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
	logChan = make(chan LogData, 1000) // 로그 데이터를 전달하기 위한 채널
)

// LogData는 로깅에 필요한 데이터를 담는 구조체
type LogData struct {
	OriginalPath  string
	StartTime     time.Time
	StatusCode    int
	Method        string
	MethodColor   string
	StatusColor   string
	Duration      time.Duration
	ClientIP      string
	UserAgent     string
	Written       int
	QueryParams   string
	ExcludedPaths []string
}

// init 함수에서 로깅 고루틴 시작
func init() {
	go loggerWorker()
}

// loggerWorker는 로그 채널에서 데이터를 읽어 로깅을 처리하는 고루틴
func loggerWorker() {
	for logData := range logChan {
		// 한 줄로 요청 정보 출력
		fmt.Printf("%s[%s]%s %s%d%s %s%s%s %s%.2fms%s %s%s%s %s%s%s %s%s%s %s%d bytes%s\n",
			Yellow, logData.StartTime.Format("2006-01-02 15:04:05.000"), Reset,
			logData.StatusColor, logData.StatusCode, Reset,
			logData.MethodColor, logData.Method, Reset,
			BoldBlue, float64(logData.Duration.Microseconds())/1000.0, Reset,
			Cyan, logData.ClientIP, Reset,
			Blue, logData.OriginalPath, Reset,
			Purple, truncateString(logData.UserAgent, 30), Reset,
			Green, logData.Written, Reset,
		)

		// 쿼리 파라미터가 있고, 제외 경로가 아닌 경우 상세 로깅
		if logData.QueryParams != "" {
			// 제외 경로 목록에 있는지 확인
			skipDetailedLogging := false
			for _, path := range logData.ExcludedPaths {
				if logData.OriginalPath == path || strings.HasPrefix(logData.OriginalPath, path+"/") {
					skipDetailedLogging = true
					break
				}
			}

			if !skipDetailedLogging {
				// 쿼리 파라미터를 별도 줄에 표시
				fmt.Printf("  %s쿼리 파라미터:%s\n", Yellow, Reset)
				params := strings.Split(logData.QueryParams, "&")
				for _, param := range params {
					kv := strings.SplitN(param, "=", 2)
					key := kv[0]
					value := ""
					if len(kv) > 1 {
						value = kv[1]

						// URL 디코딩 시도
						if decodedValue, err := url.QueryUnescape(value); err == nil {
							value = decodedValue
						}

						// 콤마로 구분된 배열 값 처리
						if strings.Contains(value, ",") {
							items := strings.Split(value, ",")
							if len(items) > 1 {
								fmt.Printf("    %s%s%s : %s배열(%d개)%s\n",
									Cyan, key, Reset,
									Yellow, len(items), Reset)

								// 각 항목을 별도 줄에 표시
								for i, item := range items {
									fmt.Printf("      %s[%d]%s %s%s%s\n",
										Yellow, i, Reset,
										Green, item, Reset)
								}
								continue // 기본 출력을 건너뛰고 다음 파라미터로
							}
						}
					}

					// 일반 키-값 쌍 출력
					fmt.Printf("    %s%s%s : %s%s%s\n",
						Cyan, key, Reset,
						Green, value, Reset)
				}
			}
		}
	}
}

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

		// 원본 경로와 로깅 상태를 컨텍스트에 저장
		ctx := context.WithValue(r.Context(), originalPathKey, originalPath)
		ctx = context.WithValue(ctx, requestLoggedKey, true)
		r = r.WithContext(ctx)

		// 응답 래퍼 생성
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // 기본값
		}

		// 응답 시작 시간
		reqStartTime := time.Now()

		// 다음 핸들러 호출
		next.ServeHTTP(rw, r)

		// 요청 처리 시간 계산
		duration := time.Since(reqStartTime)

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

		// 쿼리 파라미터 상세 로깅을 제외할 경로 목록
		excludedPaths := []string{
			"/ping",
		}

		// 로그 데이터 생성
		logData := LogData{
			OriginalPath:  originalPath,
			StartTime:     startTime,
			StatusCode:    rw.statusCode,
			Method:        r.Method,
			MethodColor:   methodColor,
			StatusColor:   statusColor,
			Duration:      duration,
			ClientIP:      clientIP,
			UserAgent:     userAgent,
			Written:       rw.written,
			QueryParams:   r.URL.RawQuery,
			ExcludedPaths: excludedPaths,
		}

		// 로그 채널에 데이터 전송 (비동기 처리)
		select {
		case logChan <- logData:
			// 채널에 성공적으로 전송됨
		default:
			// 채널이 가득 찼을 때는 로그를 버림 (백프레셔 처리)
			// 이 경우에는 로깅 대신 부하 상황 알림 로그를 출력할 수 있음
			// fmt.Println("로그 채널이 가득 찼습니다. 로그가 버려집니다.")
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
