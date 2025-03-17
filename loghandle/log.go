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
		AddSource: config.AddSource,
	}

	// 로그 형식에 따른 핸들러 생성
	if config.Format == FormatJSON {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	logger := &Logger{
		logger: slog.New(handler),
		config: config,
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
	l.mu.Lock()
	defer l.mu.Unlock()

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
		// 키-값 쌍 형식인지 확인
		if len(args) >= 2 && len(args)%2 == 0 && !hasFormatSpecifier(format) {
			// 첫 번째 인자가 문자열인지 확인
			if _, ok := args[0].(string); ok {
				// 키-값 쌍으로 처리
				msg = format
				slogArgs = args
			} else {
				// fmt 스타일 포맷팅
				msg = fmt.Sprintf(format, args...)
				slogArgs = []any{}
			}
		} else {
			// fmt 스타일 포맷팅
			msg = fmt.Sprintf(format, args...)
			slogArgs = []any{}
		}
	} else {
		// 포맷 지정자가 없는 경우
		msg = format
		slogArgs = args
	}

	// 소스 정보 추가 (AddSource가 false인 경우에도 직접 추가)
	if !l.config.AddSource && !hasSource {
		// 호출자 정보 가져오기 (3 프레임 위로 올라가서 실제 호출 위치 찾기)
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

// Debug는 디버그 레벨 로그를 기록합니다
func (l *Logger) Debug(format string, args ...any) {
	l.Log(LevelDebug, format, args...)
}

// Info는 정보 레벨 로그를 기록합니다
func (l *Logger) Info(format string, args ...any) {
	l.Log(LevelInfo, format, args...)
}

// Warn은 경고 레벨 로그를 기록합니다
func (l *Logger) Warn(format string, args ...any) {
	l.Log(LevelWarn, format, args...)
}

// Error는 오류 레벨 로그를 기록합니다
func (l *Logger) Error(format string, args ...any) {
	l.Log(LevelError, format, args...)
}

// Close는 로거 리소스를 정리합니다
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
}

// GetSlogLogger는 내부 slog.Logger를 반환합니다
func (l *Logger) GetSlogLogger() *slog.Logger {
	return l.logger
}

// 전역 함수들 - 싱글턴 로거를 통해 로그 기록
// 각 함수는 직접 소스 위치를 추적하여 Log 함수에 전달합니다
func Debug(format string, args ...any) {
	// 호출자 정보 가져오기
	_, file, line, ok := runtime.Caller(1)
	if ok && !GetLogger().config.AddSource {
		// 파일 경로에서 패키지 이름 추출
		pkgName := filepath.Base(filepath.Dir(file))
		funcName := filepath.Base(file)

		// 파일 이름에서 확장자 제거
		if idx := strings.LastIndex(funcName, "."); idx >= 0 {
			funcName = funcName[:idx]
		}

		// 소스 정보 추가
		newArgs := make([]any, len(args)+2)
		copy(newArgs, args)
		newArgs[len(args)] = "source"
		newArgs[len(args)+1] = fmt.Sprintf("%s/%s:%d", pkgName, funcName, line)
		GetLogger().Log(LevelDebug, format, newArgs...)
	} else {
		GetLogger().Debug(format, args...)
	}
}

func Info(format string, args ...any) {
	// 호출자 정보 가져오기
	_, file, line, ok := runtime.Caller(1)
	if ok && !GetLogger().config.AddSource {
		// 파일 경로에서 패키지 이름 추출
		pkgName := filepath.Base(filepath.Dir(file))
		funcName := filepath.Base(file)

		// 파일 이름에서 확장자 제거
		if idx := strings.LastIndex(funcName, "."); idx >= 0 {
			funcName = funcName[:idx]
		}

		// 소스 정보 추가
		newArgs := make([]any, len(args)+2)
		copy(newArgs, args)
		newArgs[len(args)] = "source"
		newArgs[len(args)+1] = fmt.Sprintf("%s/%s:%d", pkgName, funcName, line)
		GetLogger().Log(LevelInfo, format, newArgs...)
	} else {
		GetLogger().Info(format, args...)
	}
}

func Warn(format string, args ...any) {
	// 호출자 정보 가져오기
	_, file, line, ok := runtime.Caller(1)
	if ok && !GetLogger().config.AddSource {
		// 파일 경로에서 패키지 이름 추출
		pkgName := filepath.Base(filepath.Dir(file))
		funcName := filepath.Base(file)

		// 파일 이름에서 확장자 제거
		if idx := strings.LastIndex(funcName, "."); idx >= 0 {
			funcName = funcName[:idx]
		}

		// 소스 정보 추가
		newArgs := make([]any, len(args)+2)
		copy(newArgs, args)
		newArgs[len(args)] = "source"
		newArgs[len(args)+1] = fmt.Sprintf("%s/%s:%d", pkgName, funcName, line)
		GetLogger().Log(LevelWarn, format, newArgs...)
	} else {
		GetLogger().Warn(format, args...)
	}
}

func Error(format string, args ...any) {
	// 호출자 정보 가져오기
	_, file, line, ok := runtime.Caller(1)
	if ok && !GetLogger().config.AddSource {
		// 파일 경로에서 패키지 이름 추출
		pkgName := filepath.Base(filepath.Dir(file))
		funcName := filepath.Base(file)

		// 파일 이름에서 확장자 제거
		if idx := strings.LastIndex(funcName, "."); idx >= 0 {
			funcName = funcName[:idx]
		}

		// 소스 정보 추가
		newArgs := make([]any, len(args)+2)
		copy(newArgs, args)
		newArgs[len(args)] = "source"
		newArgs[len(args)+1] = fmt.Sprintf("%s/%s:%d", pkgName, funcName, line)
		GetLogger().Log(LevelError, format, newArgs...)
	} else {
		GetLogger().Error(format, args...)
	}
}
