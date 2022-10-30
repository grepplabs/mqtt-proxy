package log

import (
	"context"
	"sync"
)

// Fields Type to pass when we want to call WithFields for structured logging
type Fields map[string]interface{}

const (
	//Debug has verbose message
	Debug = "debug"
	//Info is default log level
	Info = "info"
	//Warn is for logging messages about possible issues
	Warn = "warn"
	//WithError is for logging errors
	Error = "error"
	// Panic log a message and panic.
	Panic = "panic"
	//Fatal is for logging fatal messages. The system shuts down after logging the message.
	Fatal = "fatal"
)

const (
	//TimeKey is a logger key for time
	TimeKey = "ts"
	//MessageKey is a logger key for message
	MessageKey = "msg"
	//LevelKey is a logger key for logging level
	LevelKey = "level"
	//CallerKey ia a logger key for caller/invoking function
	CallerKey = "caller"
	// ErrorKey is a logger key for message
	ErrorKey = "err"
)

const (
	// LogFormatJson is a format for json logging
	LogFormatJson = "json"
	// LogFormatPlain is a format for plain-text logging
	LogFormatPlain = "plain"
	// LogFormatLogfmt is a format for logfmt logging
	LogFormatLogfmt = "logfmt"
)

const (
	// ContextLogTag is used to identifier fields, which should be appended to a log entry, from a passed context
	ContextLogTag string = "logging"
)

// Logger is our contract for the logger
type Logger interface {
	Print(message string)

	Printf(format string, args ...interface{})

	Debug(message string)

	Debugf(format string, args ...interface{})

	Info(message string)

	Infof(format string, args ...interface{})

	Warn(message string)

	Warnf(format string, args ...interface{})

	Error(message string)

	Errorf(format string, args ...interface{})

	Panic(message string)

	Panicf(format string, args ...interface{})

	Fatal(message string)

	Fatalf(format string, args ...interface{})

	WithFields(keyValues Fields) Logger

	WithField(key, value string) Logger

	WithError(err error) Logger

	IsDebug() bool

	IsInfo() bool

	IsWarn() bool

	IsError() bool

	IsPanic() bool

	IsFatal() bool

	WithContext(context context.Context) Logger
}

type LogFieldNames struct {
	Time    string
	Message string
	Level   string
	Caller  string
	Error   string
}

// LogConfig stores the config for the logger
type LogConfig struct {
	LogFormat     string
	LogLevel      string
	LogFieldNames LogFieldNames
}

// NewLogger returns an instance of logger
func NewLogger(config LogConfig) Logger {
	return newZapLogger(config)
}

var (
	instance Logger
	once     sync.Once
)

// InitInstance initialize logger which will be returned by GetInstance
func InitInstance(logger Logger) {
	once.Do(func() {
		instance = logger
	})
}

func GetInstance() Logger {
	once.Do(func() {
		if instance == nil {
			instance = NewDefaultLogger()
		}
	})
	return instance
}

// NewDefaultLogger returns an instance of logger with default parameters
func NewDefaultLogger() Logger {
	config := LogConfig{
		LogFormat: LogFormatLogfmt,
		LogLevel:  Info,
	}
	return newZapLogger(config)
}

func Printf(format string, args ...interface{}) {
	GetInstance().Printf(format, args...)
}
