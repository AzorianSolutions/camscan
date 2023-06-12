package logging

import (
	"fmt"
	"time"
)

const LogLevelCritical = 100
const LogLevelError = 80
const LogLevelWarning = 60
const LogLevelInfo = 40
const LogLevelDebug = 20
const LogLevelTrace = 10
const LogLevelTrace1 = 9
const LogLevelTrace2 = 8
const LogLevelTrace3 = 7
const LogLevelTrace4 = 6
const LogLevelTrace5 = 5
const LogLevelTrace6 = 4
const LogLevelTrace7 = 3
const LogLevelTrace8 = 2
const LogLevelTrace9 = 1
const DefaultLogLevel = LogLevelInfo

var LogLevelLabels = map[int]string{
	LogLevelCritical: "CRITICAL",
	LogLevelError:    "ERROR",
	LogLevelWarning:  "WARNING",
	LogLevelInfo:     "INFO",
	LogLevelDebug:    "DEBUG",
	LogLevelTrace:    "TRACE",
	LogLevelTrace1:   "TRACE1",
	LogLevelTrace2:   "TRACE2",
	LogLevelTrace3:   "TRACE3",
	LogLevelTrace4:   "TRACE4",
	LogLevelTrace5:   "TRACE5",
	LogLevelTrace6:   "TRACE6",
	LogLevelTrace7:   "TRACE7",
	LogLevelTrace8:   "TRACE8",
	LogLevelTrace9:   "TRACE9",
}

var logLevel = DefaultLogLevel

func GetLogLevel() int {
	return logLevel
}

func SetLogLevel(level int) {
	logLevel = level
}

func Log(level int, message string, args ...interface{}) {
	if level >= logLevel {
		timestamp := time.Now().Format("2006-01-02T15:04:05.000000000Z07:00")
		label := LogLevelLabels[level]
		formatted := timestamp + " " + label + " " + message + "\n"
		fmt.Printf(formatted, args...)
	}
}

func Critical(message string, args ...interface{}) {
	Log(LogLevelCritical, message, args...)
}

func Error(message string, args ...interface{}) {
	Log(LogLevelError, message, args...)
}

func Warning(message string, args ...interface{}) {
	Log(LogLevelWarning, message, args...)
}

func Info(message string, args ...interface{}) {
	Log(LogLevelInfo, message, args...)
}

func Debug(message string, args ...interface{}) {
	Log(LogLevelDebug, message, args...)
}

func Trace(message string, args ...interface{}) {
	Log(LogLevelTrace, message, args...)
}

func Trace1(message string, args ...interface{}) {
	Log(LogLevelTrace1, message, args...)
}

func Trace2(message string, args ...interface{}) {
	Log(LogLevelTrace2, message, args...)
}

func Trace3(message string, args ...interface{}) {
	Log(LogLevelTrace3, message, args...)
}

func Trace4(message string, args ...interface{}) {
	Log(LogLevelTrace4, message, args...)
}

func Trace5(message string, args ...interface{}) {
	Log(LogLevelTrace5, message, args...)
}

func Trace6(message string, args ...interface{}) {
	Log(LogLevelTrace6, message, args...)
}

func Trace7(message string, args ...interface{}) {
	Log(LogLevelTrace7, message, args...)
}

func Trace8(message string, args ...interface{}) {
	Log(LogLevelTrace8, message, args...)
}

func Trace9(message string, args ...interface{}) {
	Log(LogLevelTrace9, message, args...)
}
