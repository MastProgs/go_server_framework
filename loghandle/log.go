package loghandle

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"runtime"
	"strings"

	"sync"

	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// LogLevel은 로그 레벨을 정의합니다
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// LogFormat은 로그 출력 형식을 정의합니다
type LogFormat int

const (
	FormatText LogFormat = iota
	FormatJSON
)

// LogOutput은 로그 출력 대상을 정의합니다
type LogOutput int

const (
	OutputConsole LogOutput = 1 << iota
	OutputFile
)

// LogConfig는 로그 설정을 정의합니다
type LogConfig struct {
	// 로그 레벨
	Level LogLevel
	// 로그 형식
	Format LogFormat
	// 로그 출력 대상 (비트 마스크로 여러 개 선택 가능)
	Output int
	// 파일 로그 설정
	FileConfig *FileLogConfig
	// 소스 코드 위치 정보 추가 여부
	AddSource bool
}

// FileLogConfig는 파일 로그 설정을 정의합니다
type FileLogConfig struct {
	// 로그 파일 경로
	Filename string
	// 최대 파일 크기 (MB)
	MaxSize int
	// 보관할 백업 파일 수
	MaxBackups int
	// 보관 기간 (일)
	MaxAge int
	// 압축 여부
	Compress bool
}

// Logger는 로그 핸들러를 정의합니다
type Logger struct {
	logger *slog.Logger
	config LogConfig
	mu     sync.Mutex

	// 비동기 로깅을 위한 필드 추가
	asyncEnabled bool
	logChan      chan logMessage
	done         chan struct{}
	wg           sync.WaitGroup
}

// logMessage는 비동기 처리를 위한 로그 메시지 구조체입니다
type logMessage struct {
	level     LogLevel
	format    string
	args      []any
	timestamp time.Time // 로그 생성 시간
}

// 싱글턴 인스턴스와 뮤텍스
var (
	instance *Logger
	mu       sync.Mutex
)

// GetLogger는 싱글턴 로거 인스턴스를 반환합니다
func GetLogger() *Logger {
	if instance == nil {
		// 기본 설정으로 로거 초기화
		InitLogger(LogConfig{
			Level:     LevelInfo,
			Format:    FormatJSON,
			Output:    int(OutputConsole),
			AddSource: false,
		})
	}
	return instance
}

// InitLogger는 로거를 초기화합니다
func InitLogger(config LogConfig) *Logger {
	mu.Lock()
	defer mu.Unlock()

	// 이미 초기화된 로거가 있으면 닫기
	if instance != nil {
		instance.Close()
	}

	// 새 로거 생성
	instance = newLogger(config)
	return instance
}

// newLogger는 새로운 로그 핸들러를 생성합니다 (내부 함수)
func newLogger(config LogConfig) *Logger {
	var handler slog.Handler
	var writers []io.Writer

	// 콘솔 출력 설정
	if config.Output&int(OutputConsole) != 0 {
		writers = append(writers, os.Stdout)
	}

	// 파일 출력 설정
	if config.Output&int(OutputFile) != 0 && config.FileConfig != nil {
		// 디렉토리 확인 및 생성
		dir := filepath.Dir(config.FileConfig.Filename)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Printf("로그 디렉토리 생성 실패: %v\n", err)
			}
		}

		fileWriter := &lumberjack.Logger{
			Filename:   config.FileConfig.Filename,
			MaxSize:    config.FileConfig.MaxSize,
			MaxBackups: config.FileConfig.MaxBackups,
			MaxAge:     config.FileConfig.MaxAge,
			Compress:   config.FileConfig.Compress,
		}
		writers = append(writers, fileWriter)
	}

	// 멀티 라이터 생성
	var writer io.Writer
	if len(writers) > 1 {
		writer = io.MultiWriter(writers...)
	} else if len(writers) == 1 {
		writer = writers[0]
	} else {
		// 기본값으로 콘솔 출력
		writer = os.Stdout
	}

	// 핸들러 옵션 설정
	opts := &slog.HandlerOptions{
		Level:     getSlogLevel(config.Level),
		AddSource: false, // 소스 정보는 항상 직접 처리하므로 비활성화
		// 기본 time 속성만 제거하는 ReplaceAttr 함수
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 최상위 레벨의 속성만 처리 (groups가 비어있으면 최상위 속성)
			if len(groups) == 0 && a.Key == "time" {
				return slog.Attr{} // 빈 속성 반환하여 제거
			}
			return a // 다른 속성은 그대로 유지
		},
	}

	// 로그 형식에 따른 핸들러 생성
	if config.Format == FormatJSON {
		// 기본 JSON 핸들러 사용
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		// 텍스트 핸들러
		handler = slog.NewTextHandler(writer, opts)
	}

	logger := &Logger{
		logger: slog.New(handler),
		config: config,

		// 비동기 로깅 기본 비활성화
		asyncEnabled: false,
		logChan:      nil,
		done:         nil,
	}

	return logger
}

// getSlogLevel은 내부 로그 레벨을 slog 레벨로 변환합니다
func getSlogLevel(level LogLevel) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Log는 지정된 레벨로 로그를 기록합니다
func (l *Logger) Log(level LogLevel, format string, args ...any) {
	// 비동기 모드 확인을 위한 락
	l.mu.Lock()
	async := l.asyncEnabled
	l.mu.Unlock()

	// 로그 생성 시간 기록
	now := time.Now()

	// 소스 정보가 이미 있는지 확인
	hasSource := false
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok && key == "source" {
				hasSource = true
				break
			}
		}
	}

	// 소스 정보가 없는 경우에만 추가
	if !hasSource {
		// 호출자 정보 가져오기
		_, file, line, ok := runtime.Caller(2) // 호출 스택 고려하여 조정
		if ok {
			// 파일 경로에서 패키지 이름 추출
			pkgName := filepath.Base(filepath.Dir(file))
			funcName := filepath.Base(file)

			// 파일 이름에서 확장자 제거
			if idx := strings.LastIndex(funcName, "."); idx >= 0 {
				funcName = funcName[:idx]
			}

			// 소스 정보를 args에 추가
			newArgs := make([]any, len(args)+2)
			copy(newArgs, args)
			newArgs[len(args)] = "source"
			newArgs[len(args)+1] = fmt.Sprintf("%s/%s:%d", pkgName, funcName, line)
			args = newArgs
		}
	}

	if async {
		// 비동기 모드면 채널에 메시지 전송
		select {
		case l.logChan <- logMessage{
			level:     level,
			format:    format,
			args:      args,
			timestamp: now, // 미리 저장한 시간 사용
		}:
			// 메시지가 성공적으로 큐에 추가됨
		default:
			// 채널이 가득 찼을 때의 처리 - 동기식으로 폴백하거나 경고 로그
			// 중요 로그를 놓치지 않기 위해 동기식으로 처리
			l.mu.Lock()
			l.processLog(level, format, args, now)
			l.mu.Unlock()
		}
	} else {
		// 동기 모드면 직접 처리
		l.mu.Lock()
		l.processLog(level, format, args, now)
		l.mu.Unlock()
	}
}

// EnableAsync는 비동기 로깅을 활성화합니다
func (l *Logger) EnableAsync(queueSize int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 이미 활성화되어 있으면 무시
	if l.asyncEnabled {
		return
	}

	// 채널 초기화
	l.logChan = make(chan logMessage, queueSize)
	l.done = make(chan struct{})
	l.asyncEnabled = true

	// 워커 고루틴 시작
	l.wg.Add(1)
	go l.logWorker()
}

// DisableAsync는 비동기 로깅을 비활성화하고 모든 보류 중인 로그를 처리합니다
func (l *Logger) DisableAsync() {
	l.mu.Lock()

	// 비동기 모드가 아니면 무시
	if !l.asyncEnabled {
		l.mu.Unlock()
		return
	}

	// 종료 신호 보내기
	close(l.done)
	l.asyncEnabled = false
	l.mu.Unlock()

	// 모든 워커가 종료될 때까지 대기
	l.wg.Wait()
}

// logWorker는 백그라운드에서 로그 메시지를 처리합니다
func (l *Logger) logWorker() {
	defer l.wg.Done()

	for {
		select {
		case msg, ok := <-l.logChan:
			if !ok {
				return
			}
			// 동기식 로그 처리 함수 호출
			l.processLog(msg.level, msg.format, msg.args, msg.timestamp)
		case <-l.done:
			// 종료 신호를 받았으므로 남은 로그 처리 후 종료
			for {
				select {
				case msg, ok := <-l.logChan:
					if !ok {
						return
					}
					l.processLog(msg.level, msg.format, msg.args, msg.timestamp)
				default:
					// 모든 메시지가 처리됨
					return
				}
			}
		}
	}
}

// processLog는 실제 로그 처리를 담당합니다 (내부 함수)
func (l *Logger) processLog(level LogLevel, format string, args []any, timestamp time.Time) {
	var msg string
	var slogArgs []any

	// 소스 정보가 이미 있는지 확인
	hasSource := false
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok && key == "source" {
				hasSource = true
				break
			}
		}
	}

	// 포맷 문자열 처리
	if hasFormatSpecifier(format) && len(args) > 0 {
		if len(args) >= 2 && len(args)%2 == 0 && !hasFormatSpecifier(format) {
			if _, ok := args[0].(string); ok {
				msg = format
				slogArgs = args
			} else {
				msg = fmt.Sprintf(format, args...)
				slogArgs = []any{}
			}
		} else {
			msg = fmt.Sprintf(format, args...)
			slogArgs = []any{}
		}
	} else {
		msg = format
		slogArgs = args
	}

	// 소스 정보 추가
	// AddSource가 false인 경우에만 여기서 직접 처리
	// AddSource가 true인 경우 ReplaceAttr에서 처리됨
	if !l.config.AddSource && !hasSource {
		// 호출자 정보는 이미 추출되어 있어야 함
		// 로그 처리를 위한 로직만 포함
		// 호출 스택: 실제 호출 위치 -> Debug/Info/Warn/Error -> Log
		_, file, line, ok := runtime.Caller(3)
		if ok {
			// 파일 경로에서 패키지 이름 추출
			pkgName := filepath.Base(filepath.Dir(file))
			funcName := filepath.Base(file)

			// 파일 이름에서 확장자 제거
			if idx := strings.LastIndex(funcName, "."); idx >= 0 {
				funcName = funcName[:idx]
			}

			// 간결한 소스 정보 추가
			slogArgs = append(slogArgs, "source", fmt.Sprintf("%s/%s:%d", pkgName, funcName, line))
		}
	}

	// 저장된 타임스탬프 추가 (timestamp 필드로 추가)
	// 실제 로그 함수 호출 시점의 시간을 기록
	slogArgs = append(slogArgs, "timestamp", timestamp.Format(time.RFC3339Nano))

	// slog 로거로 로그 기록
	switch level {
	case LevelDebug:
		l.logger.Debug(msg, slogArgs...)
	case LevelInfo:
		l.logger.Info(msg, slogArgs...)
	case LevelWarn:
		l.logger.Warn(msg, slogArgs...)
	case LevelError:
		l.logger.Error(msg, slogArgs...)
	}
}

// hasFormatSpecifier는 문자열에 포맷 지정자가 있는지 확인합니다
func hasFormatSpecifier(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+1 < len(s) {
			switch s[i+1] {
			case 's', 'd', 'v', 'f', 't', 'b', 'c', 'q', 'x', 'X', 'U', 'e', 'E', 'g', 'G', 'p':
				return true
			}
		}
	}
	return false
}

// Debug는 디버그 레벨로 로그를 기록합니다
func Debug(format string, args ...any) {
	// 호출자 정보 가져오기
	_, file, line, ok := runtime.Caller(1)
	if ok {
		// 파일 경로에서 패키지 이름 추출
		pkgName := filepath.Base(filepath.Dir(file))
		funcName := filepath.Base(file)

		// 파일 이름에서 확장자 제거
		if idx := strings.LastIndex(funcName, "."); idx >= 0 {
			funcName = funcName[:idx]
		}

		// 소스 정보를 args에 추가
		newArgs := make([]any, len(args)+2)
		copy(newArgs, args)
		newArgs[len(args)] = "source"
		newArgs[len(args)+1] = fmt.Sprintf("%s/%s:%d", pkgName, funcName, line)

		// 전역 로거로 로그 출력
		GetLogger().Log(LevelDebug, format, newArgs...)
		return
	}

	// 호출자 정보를 얻지 못한 경우 원래 인수로 로깅
	GetLogger().Log(LevelDebug, format, args...)
}

// Info는 정보 레벨로 로그를 기록합니다
func Info(format string, args ...any) {
	// 호출자 정보 가져오기
	_, file, line, ok := runtime.Caller(1)
	if ok {
		// 파일 경로에서 패키지 이름 추출
		pkgName := filepath.Base(filepath.Dir(file))
		funcName := filepath.Base(file)

		// 파일 이름에서 확장자 제거
		if idx := strings.LastIndex(funcName, "."); idx >= 0 {
			funcName = funcName[:idx]
		}

		// 소스 정보를 args에 추가
		newArgs := make([]any, len(args)+2)
		copy(newArgs, args)
		newArgs[len(args)] = "source"
		newArgs[len(args)+1] = fmt.Sprintf("%s/%s:%d", pkgName, funcName, line)

		// 전역 로거로 로그 출력
		GetLogger().Log(LevelInfo, format, newArgs...)
		return
	}

	// 호출자 정보를 얻지 못한 경우 원래 인수로 로깅
	GetLogger().Log(LevelInfo, format, args...)
}

// Warn은 경고 레벨로 로그를 기록합니다
func Warn(format string, args ...any) {
	// 호출자 정보 가져오기
	_, file, line, ok := runtime.Caller(1)
	if ok {
		// 파일 경로에서 패키지 이름 추출
		pkgName := filepath.Base(filepath.Dir(file))
		funcName := filepath.Base(file)

		// 파일 이름에서 확장자 제거
		if idx := strings.LastIndex(funcName, "."); idx >= 0 {
			funcName = funcName[:idx]
		}

		// 소스 정보를 args에 추가
		newArgs := make([]any, len(args)+2)
		copy(newArgs, args)
		newArgs[len(args)] = "source"
		newArgs[len(args)+1] = fmt.Sprintf("%s/%s:%d", pkgName, funcName, line)

		// 전역 로거로 로그 출력
		GetLogger().Log(LevelWarn, format, newArgs...)
		return
	}

	// 호출자 정보를 얻지 못한 경우 원래 인수로 로깅
	GetLogger().Log(LevelWarn, format, args...)
}

// Error는 에러 레벨로 로그를 기록합니다
func Error(format string, args ...any) {
	// 호출자 정보 가져오기
	_, file, line, ok := runtime.Caller(1)
	if ok {
		// 파일 경로에서 패키지 이름 추출
		pkgName := filepath.Base(filepath.Dir(file))
		funcName := filepath.Base(file)

		// 파일 이름에서 확장자 제거
		if idx := strings.LastIndex(funcName, "."); idx >= 0 {
			funcName = funcName[:idx]
		}

		// 소스 정보를 args에 추가
		newArgs := make([]any, len(args)+2)
		copy(newArgs, args)
		newArgs[len(args)] = "source"
		newArgs[len(args)+1] = fmt.Sprintf("%s/%s:%d", pkgName, funcName, line)

		// 전역 로거로 로그 출력
		GetLogger().Log(LevelError, format, newArgs...)
		return
	}

	// 호출자 정보를 얻지 못한 경우 원래 인수로 로깅
	GetLogger().Log(LevelError, format, args...)
}

// Close는 로거 리소스를 정리합니다
func (l *Logger) Close() {
	// 비동기 로거 종료
	l.DisableAsync()

	l.mu.Lock()
	defer l.mu.Unlock()

	// 추가 리소스 정리 로직이 필요하면 여기에 추가
}

// GetSlogLogger는 내부 slog.Logger를 반환합니다
func (l *Logger) GetSlogLogger() *slog.Logger {
	return l.logger
}

// InitLoggerWithAsync는 비동기 로깅을 활성화하여 로거를 초기화합니다
func InitLoggerWithAsync(config LogConfig, queueSize int) *Logger {
	logger := InitLogger(config)
	logger.EnableAsync(queueSize)
	return logger
}
