package logger

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
