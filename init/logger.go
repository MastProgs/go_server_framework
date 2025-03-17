package init

import (
	"fmt"
	"go_server_framework/config"
	"go_server_framework/loghandle"
	"os"
	"path/filepath"
	"strings"
)

// InitLogger는 애플리케이션 로거를 초기화합니다
func InitLogger() {
	// 설정 로드
	cfg := config.GetConfig()

	// 로그 레벨 변환
	var logLevel loghandle.LogLevel
	switch strings.ToLower(cfg.Log.Level) {
	case "debug":
		logLevel = loghandle.LevelDebug
	case "info":
		logLevel = loghandle.LevelInfo
	case "warn":
		logLevel = loghandle.LevelWarn
	case "error":
		logLevel = loghandle.LevelError
	default:
		logLevel = loghandle.LevelInfo
	}

	// 로그 포맷 변환
	var logFormat loghandle.LogFormat
	switch strings.ToLower(cfg.Log.Format) {
	case "json":
		logFormat = loghandle.FormatJSON
	case "text":
		logFormat = loghandle.FormatText
	default:
		logFormat = loghandle.FormatJSON
	}

	// 로그 출력 대상 변환
	var logOutput int
	for _, output := range cfg.Log.Output {
		switch strings.ToLower(output) {
		case "console":
			logOutput |= int(loghandle.OutputConsole)
		case "file":
			logOutput |= int(loghandle.OutputFile)
		}
	}

	// 로그 설정
	logConfig := loghandle.LogConfig{
		Level:     logLevel,
		Format:    logFormat,
		Output:    logOutput,
		AddSource: cfg.Log.AddSource,
	}

	// 파일 로그 설정
	if logOutput&int(loghandle.OutputFile) != 0 {
		// 로그 디렉토리 생성
		logDir := filepath.Dir(cfg.Log.File.Filename)
		if logDir != "." && logDir != "" {
			if err := os.MkdirAll(logDir, 0755); err != nil {
				// 아직 로거가 초기화되지 않았으므로 fmt로 출력
				fmt.Printf("로그 디렉토리 생성 실패: %v\n", err)
			}
		}

		logConfig.FileConfig = &loghandle.FileLogConfig{
			Filename:   cfg.Log.File.Filename,
			MaxSize:    cfg.Log.File.MaxSize,
			MaxBackups: cfg.Log.File.MaxBackups,
			MaxAge:     cfg.Log.File.MaxAge,
			Compress:   cfg.Log.File.Compress,
		}
	}

	// 로거 초기화 (싱글턴)
	loghandle.InitLogger(logConfig)
}
