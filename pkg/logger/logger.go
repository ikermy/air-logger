package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// LogLevel определяет уровень логирования
type LogLevel int

// RotatingWriter реализует ротацию лог-файлов без внешних зависимостей
type RotatingWriter struct {
	Filename   string
	MaxSize    int // MB
	MaxBackups int
	MaxAge     int // дни
	Compress   bool
	mu         sync.Mutex
	file       *os.File
	size       int64
}

// LoggerConfig конфигурирует логгер с использованием builder pattern
type LoggerConfig struct {
	filepath   string
	maxSize    int
	maxBackups int
	maxAge     int
	compress   bool
	minLevel   LogLevel
	caller     bool
	stdoutOnly bool
}

var generalWriter io.Writer
var logFile *RotatingWriter
var currentLogLevel LogLevel = INFO // Уровень логирования по умолчанию
var currentUseCaller = true
var timeCacheMu sync.Mutex
var timeCacheUnix int64
var timeCacheValue string

func defaultConfig() *LoggerConfig {
	return &LoggerConfig{
		maxSize:    1,
		maxBackups: 3,
		maxAge:     30,
		compress:   true,
		minLevel:   INFO, // По умолчанию отключены DEBUG логи
		caller:     true,
	}
}

// SetPatch инициализирует конфигуратор файлового логгера и возвращает конфиг для fluent API
func SetPatch(path string) *LoggerConfig {
	config := defaultConfig()
	config.filepath = path
	return config
}

// StdOut инициализирует конфигуратор логгера только для stdout
func StdOut() *LoggerConfig {
	return &LoggerConfig{
		minLevel:   INFO,
		caller:     true,
		stdoutOnly: true,
	}
}

// FromString вспомогательный хелпер string to LogLevel
func (lc *LoggerConfig) FromString(level string) LogLevel {
	switch level {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn", "warning":
		return WARNING
	case "error":
		return ERROR
	default:
		return INFO
	}
}

// WithLogLevel устанавливает минимальный уровень логирования
func (lc *LoggerConfig) WithLogLevel(level LogLevel) *LoggerConfig {
	lc.minLevel = level
	return lc
}

// WithMaxSize устанавливает максимальный размер файла в МБ
func (lc *LoggerConfig) WithMaxSize(mb int) *LoggerConfig {
	lc.maxSize = mb
	return lc
}

// WithMaxBackups устанавливает максимальное количество резервных копий
func (lc *LoggerConfig) WithMaxBackups(count int) *LoggerConfig {
	lc.maxBackups = count
	return lc
}

// WithMaxAge устанавливает максимальный возраст файлов в днях
func (lc *LoggerConfig) WithMaxAge(days int) *LoggerConfig {
	lc.maxAge = days
	return lc
}

// WithCompress включает/отключает сжатие архивов
func (lc *LoggerConfig) WithCompress(compress bool) *LoggerConfig {
	lc.compress = compress
	return lc
}

// WithCaller включает/отключает добавление file:line в логах
func (lc *LoggerConfig) WithCaller(enabled bool) *LoggerConfig {
	lc.caller = enabled
	return lc
}

// Apply применяет конфигурацию и инициализирует логгер
func (lc *LoggerConfig) Apply() {
	currentLogLevel = lc.minLevel
	currentUseCaller = lc.caller

	if logFile != nil {
		_ = logFile.close()
		logFile = nil
	}

	if lc.stdoutOnly {
		generalWriter = os.Stdout
		return
	}

	logFile = &RotatingWriter{
		Filename:   lc.filepath,
		MaxSize:    lc.maxSize,
		MaxBackups: lc.maxBackups,
		MaxAge:     lc.maxAge,
		Compress:   lc.compress,
	}

	if err := logFile.openFile(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		logFile.file = nil
	}
	generalWriter = io.MultiWriter(os.Stdout, logFile)
}

// Close корректно закрывает файл логов
func Close() {
	if logFile != nil {
		_ = logFile.close()
	}
}

func fastBase(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}
