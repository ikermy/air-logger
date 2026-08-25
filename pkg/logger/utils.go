package logger

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// GetUserLogs утилита для выборки логов
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

func extractUserID(args []interface{}) (uint32, bool, []interface{}) {
	if len(args) > 0 {
		if uid, ok := args[len(args)-1].(uint32); ok {
			return uid, true, args[:len(args)-1]
		}
	}
	return 0, false, args
}
