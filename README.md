# Go 서버 프레임워크

이 프레임워크는 Go 언어를 사용하여 빠르고 안정적인 웹 서버를 구축하기 위한 도구입니다. 
표준 라이브러리와 몇 가지 검증된 외부 패키지를 기반으로 성능과 확장성을 최적화했습니다.

## 목차

- [주요 기능](#주요-기능)
- [시작하기](#시작하기)
- [로깅 시스템](#로깅-시스템)
  - [로거 초기화](#로거-초기화)
  - [로그 레벨](#로그-레벨)
  - [로그 포맷](#로그-포맷)
  - [로그 대상](#로그-대상)
  - [소스 위치 정보](#소스-위치-정보)
  - [비동기 로깅](#비동기-로깅)
  - [예제](#로깅-예제)
- [라우터 시스템](#라우터-시스템)
  - [라우터 초기화](#라우터-초기화)
  - [엔드포인트 등록](#엔드포인트-등록)
  - [미들웨어](#미들웨어)
  - [요청 처리](#요청-처리)
  - [응답 작성](#응답-작성)
  - [예제](#라우터-예제)
- [서버설정 시스템](#서버설정-시스템)
- [에러 처리](#에러-처리)
- [프로젝트 구조](#프로젝트-구조)
- [기여 가이드](#기여-가이드)
- [라이센스](#라이센스)

## 주요 기능

- **고성능 라우터**: 빠른 URL 라우팅과 요청 처리
- **로깅 시스템**: 구조화된 로깅, 다양한 출력 옵션, 비동기 처리 지원
- **미들웨어 지원**: 인증, 로깅, 요청/응답 변환을 위한 미들웨어 체인
- **서버설정 관리**: TOML 기반 구성 파일 시스템
- **에러 처리**: 일관된 에러 처리 및 사용자 정의 가능한 에러 응답
- **보안 기능**: CORS, CSRF 방지, 입력 검증 지원

## 시작하기

### 기본 서버 실행

```go
package services

import (
	"go_server_framework/loghandle"
	"go_server_framework/router"
	"go_server_framework/services/test"
)

// RegisterAllServices는 모든 서비스를 등록합니다
func RegisterAllServices(manager *router.RouterManager) {
	// 테스트 서비스 등록
	test.RegisterRoutes(manager)
	loghandle.Info("테스트 서비스 등록 완료")

	// 여기에 추가 서비스 등록
	// 예: user.RegisterRoutes(manager)
	// 예: blog.RegisterRoutes(manager)
	// 예: shop.RegisterRoutes(manager)
}

```

## 로깅 시스템

로깅 시스템은 `loghandle` 패키지를 통해 제공됩니다. 이 시스템은 Go의 표준 `log/slog` 패키지를 기반으로 하며, 확장 기능을 추가했습니다.

### 로거 초기화

로거는 다음과 같은 방법으로 초기화할 수 있습니다:

```go
// 기본 설정으로 초기화
logger := loghandle.GetLogger()

// 커스텀 설정으로 초기화
logConfig := loghandle.LogConfig{
    Level:     loghandle.LevelInfo,    // 로그 레벨
    Format:    loghandle.FormatJSON,   // 출력 형식
    Output:    int(loghandle.OutputConsole) | int(loghandle.OutputFile), // 출력 대상
    AddSource: true,                   // 소스 코드 위치 정보 추가
    FileConfig: &loghandle.FileLogConfig{
        Filename:   "logs/app.log",    // 로그 파일 경로
        MaxSize:    10,                // 최대 파일 크기 (MB)
        MaxBackups: 5,                 // 보관할 백업 파일 수
        MaxAge:     30,                // 보관 기간 (일)
        Compress:   true,              // 로그 파일 압축 여부
    },
}
logger := loghandle.InitLogger(logConfig)
```

### 로그 레벨

로그 레벨은 다음 네 가지를 지원합니다:

- `LevelDebug`: 디버그 정보 (개발 중 상세한 정보)
- `LevelInfo`: 일반 정보 (정상적인 애플리케이션 이벤트)
- `LevelWarn`: 경고 (잠재적 문제)
- `LevelError`: 오류 (실패한 작업)

### 로그 포맷

두 가지 로그 형식을 지원합니다:

- `FormatText`: 사람이 읽기 쉬운 텍스트 형식
- `FormatJSON`: 기계가 파싱하기 쉬운 JSON 형식 (로그 수집 시스템과 통합에 적합)

### 로그 대상

로그는 다음 대상으로 출력할 수 있습니다:

- `OutputConsole`: 표준 출력 (콘솔)
- `OutputFile`: 파일 (순환 로깅 지원)

비트 OR 연산자를 사용하여 여러 대상을 결합할 수 있습니다:
```go
Output: int(loghandle.OutputConsole) | int(loghandle.OutputFile)
```

### 소스 위치 정보

`AddSource` 옵션을 `true`로 설정하면 로그 항목에 소스 파일 및 행 번호가 포함됩니다:

```json
{"level":"INFO","msg":"라우터 초기화 완료","source":"init/router:28","timestamp":"2025-03-18T03:56:52.3621425+09:00"}
```

### 비동기 로깅

성능이 중요한 애플리케이션의 경우 비동기 로깅을 활성화할 수 있습니다:

```go
// 기본 로거 초기화 후 비동기 모드 활성화
loghandle.InitLogger(logConfig)
loghandle.GetLogger().EnableAsync(1000) // 버퍼 크기 1000

// 또는 한 번에 비동기 로거 초기화
logger := loghandle.InitLoggerWithAsync(logConfig, 1000)
```

비동기 로깅의 장점:
- 로깅 작업이 메인 스레드를 차단하지 않음
- 성능 향상, 특히 I/O 바인딩 작업에서

주의사항:
- 로그 메시지가 즉시 기록되지 않을 수 있음
- 버퍼가 꽉 차면 새 로그 메시지가 동기적으로 처리됨

비고:
- 비동기여도 로깅 첫 함수 호출 시점으로 timestamp 가 작성됨
- 시각이 기록된 이후, 로깅 큐에 추가되는 형태

애플리케이션을 종료하기 전에 비동기 로거를 비활성화하는 것이 좋습니다:
```go
loghandle.GetLogger().DisableAsync()
```

### 로깅 예제

네 가지 로그 레벨 사용 방법:

```go
// 패키지 레벨 함수 사용
loghandle.Debug("디버그 메시지: %s", "상세 정보")
loghandle.Info("정보 메시지: %s", "일반 정보")
loghandle.Warn("경고 메시지: %s", "잠재적 문제")
loghandle.Error("오류 메시지: %s", "실패 내용")

// 또는 로거 인스턴스 직접 사용
logger := loghandle.GetLogger()
logger.Debug("디버그 메시지")
logger.Info("정보 메시지")
logger.Warn("경고 메시지")
logger.Error("오류 메시지")

// 구조화된 로깅
loghandle.Info("사용자 로그인", 
    "user_id", 12345, 
    "ip", "192.168.1.1", 
    "status", "success")
```

로그 출력 예시 (JSON 형식):
```json
{"level":"INFO","msg":"서버 시작됨","source":"main:25","timestamp":"2025-03-18T03:56:52.3621425+09:00"}
{"level":"INFO","msg":"사용자 로그인","user_id":12345,"ip":"192.168.1.1","status":"success","source":"auth/handler:42","timestamp":"2025-03-18T03:56:55.1234567+09:00"}
```

## 라우터 시스템

라우터 시스템은 HTTP 요청을 처리하기 위한 경량 프레임워크를 제공합니다.

### 라우터 초기화

```go
// 기본 라우터 생성
r := router.NewRouter()

// 또는 맞춤 설정으로 생성
r := router.NewRouterWithConfig(router.RouterConfig{
    TrustedProxies: []string{"127.0.0.1"},
    MaxRequestSize: 1024 * 1024 * 10, // 10MB
})
```

### 엔드포인트 등록

다양한 HTTP 메서드에 대한 핸들러를 등록할 수 있습니다:

```go
// 기본 라우트
r.GET("/users", listUsers)
r.POST("/users", createUser)
r.PUT("/users/:id", updateUser)
r.DELETE("/users/:id", deleteUser)

// 그룹화된 라우트
api := r.Group("/api")
{
    v1 := api.Group("/v1")
    {
        v1.GET("/products", listProducts)
        v1.POST("/products", createProduct)
    }
}

// 핸들러 함수 정의
func listUsers(c *router.Context) {
    c.JSON(200, map[string]interface{}{
        "users": []string{"user1", "user2"},
    })
}
```

### 미들웨어

미들웨어를 사용하여, 요청 처리 파이프라인에 추가 기능을 넣을 수 있습니다:

```go
// 전역 미들웨어
r.Use(router.Logger(), router.Recovery())

// 그룹 미들웨어
api := r.Group("/api", router.Auth())

// 라우트별 미들웨어
r.GET("/admin", adminHandler, router.AdminAuth())
```

사용자 정의 미들웨어 생성:

```go
func MyMiddleware() router.HandlerFunc {
    return func(c *router.Context) {
        // 요청 처리 전 작업
        startTime := time.Now()
        
        // 다음 핸들러 호출
        c.Next()
        
        // 요청 처리 후 작업
        duration := time.Since(startTime)
        loghandle.Info("요청 처리 시간", "path", c.Path(), "duration", duration.String())
    }
}
```

### 요청 처리

컨텍스트 객체를 통해 요청을 처리할 수 있습니다:

```go
func userHandler(c *router.Context) {
    // URL 파라미터
    id := c.Param("id")
    
    // 쿼리 파라미터
    page := c.Query("page", "1") // 기본값 1
    
    // 폼 데이터
    name := c.PostForm("name")
    
    // JSON 바디
    var user User
    if err := c.BindJSON(&user); err != nil {
        c.JSON(400, map[string]string{"error": "잘못된 요청 형식"})
        return
    }
    
    // 헤더 접근
    token := c.GetHeader("Authorization")
}
```

### 응답 작성

다양한 형식으로 응답을 보낼 수 있습니다:

```go
// JSON 응답
c.JSON(200, map[string]interface{}{
    "id": 1,
    "name": "홍길동",
})

// XML 응답
c.XML(200, someXMLStruct)

// HTML 응답
c.HTML(200, "<h1>안녕하세요</h1>")

// 텍스트 응답
c.String(200, "요청 처리 완료")

// 리다이렉트
c.Redirect(302, "/new-location")

// 파일 전송
c.File("/path/to/file.pdf")

// 헤더 설정
c.SetHeader("Content-Type", "application/json")
```

### 라우터 예제

`services/test/router.go` 파일을 참고하여주세요.

## 서버설정 시스템

TOML 기반 구성 파일을 사용하여 애플리케이션 설정을 관리합니다:

```toml
# config.toml
[server]
port = 8080
host = "0.0.0.0"
```

Go 코드에서 설정 읽기:

```go
config, err := config.LoadConfig("config.toml")
if err != nil {
    loghandle.Error("설정 파일 로드 실패", "error", err.Error())
    return
}

// 설정 사용
port := config.GetInt("server.port")
```

## 에러 처리

일관된 에러 처리를 위한 표준화된 에러 생성 및, 응답 기능을 제공합니다:

```go
import "go_server_framework/errors"

// 에러 생성
err := errors.New("오류 메시지")
err := errors.Newf("ID %d를 찾을 수 없습니다", id)
err := errors.WithCode(404, "리소스를 찾을 수 없습니다")
err := errors.WithStack(originalError) // 스택 트레이스 포함

// 컨텍스트에서 사용
if user == nil {
    return errors.NotFound("사용자를 찾을 수 없습니다")
}

// HTTP 응답 예시
func getUser(c *router.Context) {
    user, err := userService.GetUser(id)
    if err != nil {
        if errors.Is(err, errors.NotFound) {
            c.Error(404, "사용자를 찾을 수 없습니다")
            return
        }
        c.Error(500, "서버 오류")
        return
    }
    
    c.JSON(200, user)
}
```

## 프로젝트 구조

```
go_server_framework/
├── config/           # 설정 관리
│   ├── config.go     # 설정 로드/파싱
│   └── config.toml   # 기본 설정 파일
├── errors/           # 에러 처리
│   └── errors.go     # 에러 유틸리티
├── loghandle/        # 로깅 시스템
│   └── log.go        # 로그 핸들러
├── router/           # 라우팅 시스템
│   ├── context.go    # 요청/응답 컨텍스트
│   ├── middleware.go # 미들웨어 컬렉션
│   └── router.go     # 라우터 구현
├── utils/            # 유틸리티 함수
├── examples/         # 사용 예제
├── go.mod            # Go 모듈 정의
├── go.sum            # 의존성 체크섬
├── main.go           # 애플리케이션 진입점
└── README.md         # 프로젝트 문서
```

