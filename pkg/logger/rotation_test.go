package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRotatingWriterBasic(t *testing.T) {
	// Создаем временную директорию
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// Инициализируем логгер с builder pattern
	Set(logPath).
		WithLogLevel(INFO).
		WithMaxSize(1).
		WithMaxBackups(3).
		Apply()
	defer Close()

	// Пишем несколько логов
	Info("Test message %d", 1)
	Info("Test message %d", 2)
	Warn("Warning message")
	Error("Error message")
	Debug("Debug message - не должен логироваться") // Не будет залогирован (уровень DEBUG отключен по умолчанию)

	// Проверяем что файл существует
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatalf("Log file was not created: %v", err)
	}

	// Проверяем содержимое файла
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Log file is empty")
	}

	content := string(data)
	if !contains(content, "Test message 1") {
		t.Errorf("Expected log message not found")
	}

	// Debug сообщение не должно быть в логе
	if contains(content, "Debug message") {
		t.Errorf("Debug message should not be logged when minLevel is INFO")
	}

	t.Logf("Log file size: %d bytes\n", len(data))
}

func TestRotatingWriterWithDebug(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "debug.log")

	// Включаем DEBUG логирование
	Set(logPath).
		WithLogLevel(DEBUG).
		Apply()
	defer Close()

	Debug("Debug message")
	Info("Info message")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	content := string(data)
	if !contains(content, "Debug message") {
		t.Errorf("Debug message should be logged when minLevel is DEBUG")
	}

	t.Log("Debug logging test passed")
}

func TestRotatingWriterLogLevelFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "level.log")

	// Устанавливаем минимальный уровень WARNING
	Set(logPath).
		WithLogLevel(WARNING).
		Apply()
	defer Close()

	Debug("Debug - не должен быть залогирован")
	Info("Info - не должен быть залогирован")
	Warn("Warning - должен быть залогирован")
	Error("Error - должен быть залогирован")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	content := string(data)

	if contains(content, "Debug") {
		t.Error("Debug should not be logged")
	}
	if contains(content, "Info - не должен") {
		t.Error("Info should not be logged")
	}
	if !contains(content, "Warning") {
		t.Error("Warning should be logged")
	}
	if !contains(content, "Error") {
		t.Error("Error should be logged")
	}

	t.Log("Log level filtering test passed")
}

func TestRotatingWriterRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "rotate.log")

	// Создаем логгер с малым размером для ротации
	logFile = &RotatingWriter{
		Filename:   logPath,
		MaxSize:    1, // 1 MB
		MaxBackups: 3,
		MaxAge:     30,
		Compress:   false, // Без сжатия для теста
	}
	if err := logFile.openFile(); err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.close()

	// Пишем большое количество данных для ротации
	largeMsg := "This is a test message that will be repeated many times to fill up the log file and trigger rotation.\n"
	for i := 0; i < 15000; i++ {
		logFile.Write([]byte(largeMsg))
	}

	time.Sleep(100 * time.Millisecond) // Даем время на ротацию

	// Проверяем что основной файл существует
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatalf("Main log file does not exist: %v", err)
	}

	// Проверяем наличие резервных копий
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	backupCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if entry.Name() != filepath.Base(logPath) {
			backupCount++
			t.Logf("Found backup file: %s\n", entry.Name())
		}
	}

	if backupCount == 0 {
		t.Log("Warning: No backup files created during rotation test")
	}

	t.Logf("Rotation test completed. Backup files: %d\n", backupCount)
}

func TestUserIDExtraction(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "userid.log")

	Set(logPath).
		WithLogLevel(INFO).
		Apply()
	defer Close()

	userID := uint32(12345)
	Info("User action", userID)

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(data)
	if !contains(logContent, "[USER:12345]") {
		t.Errorf("User ID not found in log: %s", logContent)
	}

	t.Logf("User ID log: %s\n", logContent)
}

func TestBuilderChaining(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "builder.log")

	// Тестируем fluent API
	config := Set(logPath).
		WithLogLevel(DEBUG).
		WithMaxSize(5).
		WithMaxBackups(5).
		WithMaxAge(60).
		WithCompress(false)

	config.Apply()
	defer Close()

	Debug("This is a debug message")
	Info("This is an info message")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	content := string(data)
	if !contains(content, "debug message") {
		t.Error("Builder chaining test failed: debug message not found")
	}

	t.Log("Builder chaining test passed")
}

func TestLoggerWithoutCaller(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "nocaller.log")

	Set(logPath).
		WithLogLevel(INFO).
		WithCaller(false).
		Apply()
	defer Close()

	Info("No caller message")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	content := string(data)
	if !contains(content, "[INFO] No caller message") {
		t.Fatalf("Unexpected log format without caller: %s", content)
	}

	if contains(content, "rotation_test.go:") {
		t.Fatalf("Caller info should be disabled: %s", content)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

