package logger

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

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
	sb.WriteString(colorReset)
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
		return red
	case "[WARNING]":
		return yellow
	case "[DEBUG]":
		return green
	case "[FATAL]":
		return purple
	default:
		return white
	}
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
