package logger

const (
	// ANSI цветовые коды
	colorReset = "\033[0m"
	white      = ""         // INFO
	red        = "\033[31m" // ERROR
	yellow     = "\033[33m" // WARNING
	green      = "\033[32m" // DEBUG
	purple     = "\033[35m" // FATAL

	// Уровни логирования
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	FATAL
)
