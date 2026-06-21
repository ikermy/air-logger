package logger

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ANSI цветовые коды
const (
	ColorReset  = "\033[0m"
	ColorWhite  = ""         // INFO
	ColorRed    = "\033[31m" // ERROR
	ColorYellow = "\033[33m" // WARNING
	ColorGreen  = "\033[32m" // DEBUG
	ColorPurple = "\033[35m" // FATAL
)

// LogLevel определяет уровень логирования
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	FATAL
)

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

// --- публичные методы ---
func Infoln(args ...interface{})               { 
	if currentLogLevel <= INFO {
		logMessage("[INFO]", 2, args...) 
	}
}

func Info(format string, args ...interface{})  { 
	if currentLogLevel <= INFO {
		logMessagef(format, "[INFO]", 2, args...)
	}
}

func Error(format string, args ...interface{}) { 
	if currentLogLevel <= ERROR {
		logMessagef(format, "[ERROR]", 2, args...)
	}
}

func Warn(format string, args ...interface{})  { 
	if currentLogLevel <= WARNING {
		logMessagef(format, "[WARNING]", 2, args...)
	}
}

func Debug(format string, args ...interface{}) { 
	if currentLogLevel <= DEBUG {
		logMessagef(format, "[DEBUG]", 2, args...)
	}
}

func Fatal(args ...interface{})                { 
	logMessage("[FATAL]", 2, args...)
	os.Exit(1) 
}

func Fatalf(format string, args ...interface{}) {
	logMessagef(format, "[FATAL]", 2, args...)
	os.Exit(1)
}

// --- внутренние функции ---
func logMessagef(format, level string, skip int, args ...interface{}) {
	userID, hasUserID, args := extractUserID(args)
	message := fmt.Sprintf(format, args...)
	writeLog(level, skip+1, message, userID, hasUserID)
}

func logMessage(level string, skip int, args ...interface{}) {
	userID, hasUserID, args := extractUserID(args)

	var sb strings.Builder
	sb.Grow(len(args) * 12) // предвыделение буфера

	for i, arg := range args {
		if i > 0 {
			sb.WriteByte(' ')
		}
		switch v := arg.(type) {
		case int:
			sb.WriteString(strconv.Itoa(v))
		case int64:
			sb.WriteString(strconv.FormatInt(v, 10))
		case uint32:
			sb.WriteString(strconv.Itoa(int(v)))
		case uint64:
			sb.WriteString(strconv.FormatUint(v, 10))
		case bool:
			sb.WriteString(strconv.FormatBool(v))
		case string:
			sb.WriteString(v)
		default:
			sb.WriteString(fmt.Sprint(v)) // fallback для остальных типов
		}
	}

	writeLog(level, skip+1, sb.String(), userID, hasUserID)
}

func extractUserID(args []interface{}) (uint32, bool, []interface{}) {
	if len(args) > 0 {
		if uid, ok := args[len(args)-1].(uint32); ok {
			return uid, true, args[:len(args)-1]
		}
	}
	return 0, false, args
}

func writeLog(level string, skip int, message string, userID uint32, hasUserID bool) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("logMessage panic:", r)
		}
	}()

	now := cachedNow()
	color := levelColor(level)

	file := ""
	line := 0
	if currentUseCaller {
		_, callerFile, callerLine, ok := runtime.Caller(skip)
		if !ok {
			file = "???"
			line = 0
		} else {
			file = fastBase(callerFile)
			line = callerLine
		}
	}

	var sb strings.Builder
	sb.Grow(len(message) + len(now) + len(level) + 48) // предвыделение

	sb.WriteString(color)
	sb.WriteString(now)
	if currentUseCaller {
		sb.WriteByte(' ')
		sb.WriteString(file)
		sb.WriteByte(':')
		sb.WriteString(strconv.Itoa(line))
		sb.WriteString(": ")
	} else {
		sb.WriteString(": ")
	}
	sb.WriteString(level)
	sb.WriteByte(' ')
	if hasUserID {
		sb.WriteString("[USER:")
		sb.WriteString(strconv.FormatUint(uint64(userID), 10))
		sb.WriteString("] ")
	}
	sb.WriteString(message)
	sb.WriteString(ColorReset)
	sb.WriteByte('\n')

	if generalWriter != nil {
		_, _ = io.WriteString(generalWriter, sb.String())
	} else {
		_, _ = io.WriteString(os.Stdout, sb.String())
	}
}

func levelColor(level string) string {
	switch level {
	case "[ERROR]":
		return ColorRed
	case "[WARNING]":
		return ColorYellow
	case "[DEBUG]":
		return ColorGreen
	case "[FATAL]":
		return ColorPurple
	default:
		return ColorWhite
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

func cachedNow() string {
	now := time.Now()
	sec := now.Unix()

	timeCacheMu.Lock()
	defer timeCacheMu.Unlock()

	if timeCacheUnix != sec || timeCacheValue == "" {
		timeCacheUnix = sec
		timeCacheValue = now.Format("2006/01/02 15:04:05")
	}

	return timeCacheValue
}

// --- утилита для выборки логов ---
func GetUserLogs(logFilePath string, userID uint32, writer func(string)) error {
	logMsg := func(msg string) {
		if writer != nil {
			writer(msg)
		} else {
			fmt.Println(msg)
		}
	}

	file, err := os.Open(logFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	userPattern := fmt.Sprintf("[USER:%d]", userID)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, userPattern) {
			logMsg(line)
		}
	}
	return scanner.Err()
}

// --- методы RotatingWriter ---

// Write реализует интерфейс io.Writer для ротации лог-файлов
func (rw *RotatingWriter) Write(p []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file == nil {
		if err := rw.openFile(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			return 0, err
		}
	}

	// Проверяем, нужна ли ротация по размеру
	if rw.MaxSize > 0 && rw.size+int64(len(p)) > int64(rw.MaxSize)*1024*1024 {
		if err := rw.rotate(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to rotate log file: %v\n", err)
		}
	}

	n, err = rw.file.Write(p)
	rw.size += int64(n)
	return n, err
}

// openFile открывает лог-файл, создавая его если необходимо
func (rw *RotatingWriter) openFile() error {
	if err := os.MkdirAll(filepath.Dir(rw.Filename), 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(rw.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	rw.file = file

	// Получаем текущий размер файла
	info, err := file.Stat()
	if err == nil {
		rw.size = info.Size()
	}

	return nil
}

// rotate выполняет ротацию лог-файла
func (rw *RotatingWriter) rotate() error {
	if rw.file != nil {
		_ = rw.file.Close()
	}

	rw.size = 0

	// Переименовываем текущий файл с временной меткой
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	ext := filepath.Ext(rw.Filename)
	base := strings.TrimSuffix(rw.Filename, ext)
	backupName := fmt.Sprintf("%s-%s%s", base, timestamp, ext)

	if err := os.Rename(rw.Filename, backupName); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Сжимаем резервную копию если требуется
	if rw.Compress {
		go rw.compressFile(backupName)
	}

	// Очищаем старые файлы
	go rw.cleanupOldFiles()

	// Открываем новый файл
	return rw.openFile()
}

// compressFile сжимает файл в gzip
func (rw *RotatingWriter) compressFile(filename string) {
	source, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file for compression: %v\n", err)
		return
	}
	defer source.Close()

	compressedName := filename + ".gz"
	compressed, err := os.Create(compressedName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create compressed file: %v\n", err)
		return
	}
	defer compressed.Close()

	gzipWriter := gzip.NewWriter(compressed)
	defer gzipWriter.Close()

	if _, err := io.Copy(gzipWriter, source); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to compress file: %v\n", err)
		return
	}

	// Удаляем оригинальный файл после успешного сжатия
	_ = os.Remove(filename)
}

// cleanupOldFiles удаляет старые файлы по возрасту и количеству
func (rw *RotatingWriter) cleanupOldFiles() {
	dir := filepath.Dir(rw.Filename)
	filename := filepath.Base(rw.Filename)
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read log directory: %v\n", err)
		return
	}

	type fileInfo struct {
		name    string
		modTime time.Time
	}

	var backups []fileInfo

	// Собираем все резервные копии
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Проверяем, является ли файл резервной копией
		if !strings.HasPrefix(name, base+"-") {
			continue
		}

		if !strings.HasSuffix(name, ext) && !strings.HasSuffix(name, ext+".gz") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backups = append(backups, fileInfo{
			name:    name,
			modTime: info.ModTime(),
		})
	}

	// Сортируем по времени модификации (новые сначала)
	for i := 0; i < len(backups)-1; i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[j].modTime.After(backups[i].modTime) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	// Удаляем лишние резервные копии (более MaxBackups)
	if rw.MaxBackups > 0 && len(backups) > rw.MaxBackups {
		for i := rw.MaxBackups; i < len(backups); i++ {
			_ = os.Remove(filepath.Join(dir, backups[i].name))
		}
		backups = backups[:rw.MaxBackups]
	}

	// Удаляем старые файлы (более MaxAge дней)
	if rw.MaxAge > 0 {
		cutoff := time.Now().Add(-time.Duration(rw.MaxAge*24) * time.Hour)
		for _, backup := range backups {
			if backup.modTime.Before(cutoff) {
				_ = os.Remove(filepath.Join(dir, backup.name))
			}
		}
	}
}

// close закрывает файл логов
func (rw *RotatingWriter) close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file != nil {
		return rw.file.Close()
	}
	return nil
}
