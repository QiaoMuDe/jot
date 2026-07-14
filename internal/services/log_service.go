package services

import (
	"fmt"
	"os"
	"path/filepath"

	"gitee.com/MM-Q/fastlog"
)

// LogLevel 枚举映射
const (
	LevelDebug = iota // 0
	LevelInfo         // 1
	LevelWarn         // 2
	LevelError        // 3
	LevelFatal        // 4
	LevelPanic        // 5
)

// LogService 管理 fastlog Logger 实例
type LogService struct {
	Logger *fastlog.Logger
	dir    string
}

// NewLogService 创建 LogService 实例
func NewLogService() *LogService {
	return &LogService{}
}

// Init 初始化 Logger，日志写入 dir/app.log，level 为 fastlog.Level
func (s *LogService) Init(logDir string, level fastlog.Level) error {
	s.dir = logDir

	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	logPath := filepath.Join(logDir, "app.log")

	// 基于 Prod() 配置
	cfg := fastlog.Prod(logPath)
	cfg.Level = level // 使用传入的级别

	s.Logger = fastlog.New(cfg)
	return nil
}

// SetLevel 动态调整日志级别
func (s *LogService) SetLevel(level fastlog.Level) {
	if s.Logger != nil {
		s.Logger.SetLevel(level)
	}
}

// Close 关闭 Logger，确保缓冲区落盘
func (s *LogService) Close() {
	if s.Logger != nil {
		_ = s.Logger.Close()
	}
}

// LogDir 返回日志目录路径
func (s *LogService) LogDir() string {
	return s.dir
}

// LevelFromInt 将 int 转换为 fastlog.Level，范围 0-5，越界回退 INFO
func LevelFromInt(n int) fastlog.Level {
	switch n {
	case 0:
		return fastlog.DEBUG
	case 1:
		return fastlog.INFO
	case 2:
		return fastlog.WARN
	case 3:
		return fastlog.ERROR
	case 4:
		return fastlog.FATAL
	case 5:
		return fastlog.PANIC
	default:
		return fastlog.INFO
	}
}

// LevelToInt 将 fastlog.Level 转换为 int
func LevelToInt(l fastlog.Level) int {
	switch l {
	case fastlog.DEBUG:
		return 0
	case fastlog.INFO:
		return 1
	case fastlog.WARN:
		return 2
	case fastlog.ERROR:
		return 3
	case fastlog.FATAL:
		return 4
	case fastlog.PANIC:
		return 5
	default:
		return 1
	}
}
