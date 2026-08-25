package logger

import "os"

// --- публичные методы ---
func Infoln(args ...interface{}) {
	if currentLogLevel <= INFO {
		logMessage("[INFO]", 2, args...)
	}
}

func Info(format string, args ...interface{}) {
	if currentLogLevel <= INFO {
		logMessagef(format, "[INFO]", 2, args...)
	}
}

func Error(format string, args ...interface{}) {
	if currentLogLevel <= ERROR {
		logMessagef(format, "[ERROR]", 2, args...)
	}
}

func Warn(format string, args ...interface{}) {
	if currentLogLevel <= WARNING {
		logMessagef(format, "[WARNING]", 2, args...)
	}
}

func Debug(format string, args ...interface{}) {
	if currentLogLevel <= DEBUG {
		logMessagef(format, "[DEBUG]", 2, args...)
	}
}

func Fatal(args ...interface{}) {
	logMessage("[FATAL]", 2, args...)
	os.Exit(1)
}

func Fatalf(format string, args ...interface{}) {
	logMessagef(format, "[FATAL]", 2, args...)
	os.Exit(1)
}
