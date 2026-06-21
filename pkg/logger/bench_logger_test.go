package logger

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
)

// старая версия
func logMessageConcatOld(args ...interface{}) string {
	var parts []string
	for _, arg := range args {
		parts = append(parts, fmt.Sprintf("%v", arg))
	}
	return strings.Join(parts, " ")
}

// новая версия (Builder)
func logMessageConcatNew(args ...interface{}) string {
	var sb strings.Builder
	sb.Grow(len(args) * 12)
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
			sb.WriteString(strconv.FormatUint(uint64(v), 10))
		case uint64:
			sb.WriteString(strconv.FormatUint(v, 10))
		case bool:
			sb.WriteString(strconv.FormatBool(v))
		case string:
			sb.WriteString(v)
		default:
			sb.WriteString(fmt.Sprint(v))
		}
	}
	return sb.String()
}

// чистая конкатенация через +
func logMessageConcatPlus(args ...interface{}) string {
	if len(args) == 0 {
		return ""
	}
	s := fmt.Sprint(args[0])
	for i := 1; i < len(args); i++ {
		s += " " + fmt.Sprint(args[i])
	}
	return s
}

func BenchmarkLogMessageConcatOld(b *testing.B) {
	args := []interface{}{"user", 42, true, "some text", 3.14}
	for i := 0; i < b.N; i++ {
		_ = logMessageConcatOld(args...)
	}
}

func BenchmarkLogMessageConcatNew(b *testing.B) {
	args := []interface{}{"user", 42, true, "some text", 3.14}
	for i := 0; i < b.N; i++ {
		_ = logMessageConcatNew(args...)
	}
}

func BenchmarkLogMessageConcatPlus(b *testing.B) {
	args := []interface{}{"user", 42, true, "some text", 3.14}
	for i := 0; i < b.N; i++ {
		_ = logMessageConcatPlus(args...)
	}
}

func BenchmarkWriteLogFullPath(b *testing.B) {
	prevWriter := generalWriter
	prevCaller := currentUseCaller
	generalWriter = io.Discard
	currentUseCaller = true
	defer func() {
		generalWriter = prevWriter
		currentUseCaller = prevCaller
	}()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		writeLog("[INFO]", 0, "some text", 12345, true)
	}
}

func BenchmarkWriteLogNoCaller(b *testing.B) {
	prevWriter := generalWriter
	prevCaller := currentUseCaller
	generalWriter = io.Discard
	currentUseCaller = false
	defer func() {
		generalWriter = prevWriter
		currentUseCaller = prevCaller
	}()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		writeLog("[INFO]", 0, "some text", 12345, true)
	}
}

func BenchmarkDebugDisabled(b *testing.B) {
	prevLevel := currentLogLevel
	currentLogLevel = INFO
	defer func() { currentLogLevel = prevLevel }()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Debug("debug message %d", i)
	}
}

func BenchmarkStdOutApply(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		StdOut().
			WithLogLevel(INFO).
			WithCaller(false).
			Apply()
	}
	Close()
}

// go test -bench=BenchmarkLogMessageConcat -benchmem -run=^$
