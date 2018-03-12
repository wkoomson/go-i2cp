package go_i2cp

import "fmt"

const (
	PROTOCOL = 1 << 0
	LOGIC    = 1 << 1

	DEBUG   = 1 << 4
	INFO    = 1 << 5
	WARNING = 1 << 6
	ERROR   = 1 << 7
	FATAL   = 1 << 8

	STRINGMAP      = 1 << 9
	INTMAP         = 1 << 10
	QUEUE          = 1 << 11
	STREAM         = 1 << 12
	CRYPTO         = 1 << 13
	TCP            = 1 << 14
	CLIENT         = 1 << 15
	CERTIFICATE    = 1 << 16
	LEASE          = 1 << 17
	DESTINATION    = 1 << 18
	SESSION        = 1 << 19
	SESSION_CONFIG = 1 << 20
	TEST           = 1 << 21
	DATAGRAM       = 1 << 22
	CONFIG_FILE    = 1 << 23
	VERSION        = 1 << 24

	TAG_MASK       = 0x0000000f
	LEVEL_MASK     = 0x000001f0
	COMPONENT_MASK = 0xfffffe00

	ALL = 0xffffffff
)

type LoggerTags = uint32
type LoggerCallbacks struct {
	opaque *interface{}
	onLog  func(*Logger, LoggerTags, string)
}
type Logger struct {
	callbacks *LoggerCallbacks
	logLevel  int
}

var logInstance = &Logger{}

// TODO filter
func LogInit(callbacks *LoggerCallbacks, level int) {
	logInstance = &Logger{callbacks: callbacks}
	logInstance.setLogLevel(level)
}
func Debug(tags LoggerTags, message string, args ...interface{}) {
	logInstance.log(tags|DEBUG, message, args...)
}
func Info(tags LoggerTags, message string, args ...interface{}) {
	logInstance.log(tags|INFO, message, args...)
}
func Warning(tags LoggerTags, message string, args ...interface{}) {
	logInstance.log(tags|WARNING, message, args...)
}
func Error(tags LoggerTags, message string, args ...interface{}) {
	logInstance.log(tags|ERROR, message, args...)
}
func Fatal(tags LoggerTags, message string, args ...interface{}) {
	logInstance.log(tags|FATAL, message, args...)
}

func (l *Logger) log(tags LoggerTags, format string, args ...interface{}) {
	if l.callbacks == nil {
		fmt.Printf(format+"\n", args)
	} else {
		l.callbacks.onLog(l, tags, fmt.Sprintf(format, args))
	}
}

func (l *Logger) setLogLevel(level int) {
	switch level {
	case DEBUG:
	case INFO:
	case WARNING:
	case ERROR:
	case FATAL:
		l.logLevel = level
	default:
		l.logLevel = ERROR
	}
}
