package log

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestZapLogger(t *testing.T) {
	a := assert.New(t)

	type LevelEnabled struct {
		debug bool
		info  bool
		warn  bool
		error bool
		panic bool
		fatal bool
	}

	tt := []struct {
		name         string
		logger       Logger
		levelEnabled LevelEnabled
	}{
		{"default logger", NewDefaultLogger(), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
		{"default logger with field", NewDefaultLogger().WithField("tag", "value"), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
		{"default logger with error", NewDefaultLogger().WithError(errors.New("my error")), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
		{"default logger with nil error", NewDefaultLogger().WithError(nil), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
		{"plain debug", NewLogger(LogConfig{LogFormat: LogFormatPlain, LogLevel: "debug"}), LevelEnabled{
			debug: true, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"plain info", NewLogger(LogConfig{LogFormat: LogFormatPlain, LogLevel: "info"}), LevelEnabled{
			debug: false, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"plain warn", NewLogger(LogConfig{LogFormat: LogFormatPlain, LogLevel: "warn"}), LevelEnabled{
			debug: false, info: false, warn: true, error: true, panic: true, fatal: true}},
		{"plain error", NewLogger(LogConfig{LogFormat: LogFormatPlain, LogLevel: "error"}), LevelEnabled{
			debug: false, info: false, warn: false, error: true, panic: true, fatal: true}},
		{"plain panic", NewLogger(LogConfig{LogFormat: LogFormatPlain, LogLevel: "panic"}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: true, fatal: true}},
		{"plain fatal", NewLogger(LogConfig{LogFormat: LogFormatPlain, LogLevel: "fatal"}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: false, fatal: true}},
		{"plain with error", NewLogger(LogConfig{LogFormat: LogFormatPlain, LogLevel: "info"}).WithError(errors.New("my error")), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
		{"json debug", NewLogger(LogConfig{LogFormat: LogFormatJson, LogLevel: Debug}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: true, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"json info", NewLogger(LogConfig{LogFormat: LogFormatJson, LogLevel: Info}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: false, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"json warn", NewLogger(LogConfig{LogFormat: LogFormatJson, LogLevel: Warn}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: false, info: false, warn: true, error: true, panic: true, fatal: true}},
		{"json error", NewLogger(LogConfig{LogFormat: LogFormatJson, LogLevel: Error}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: false, info: false, warn: false, error: true, panic: true, fatal: true}},
		{"json panic", NewLogger(LogConfig{LogFormat: LogFormatJson, LogLevel: Panic}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: true, fatal: true}},
		{"json fatal", NewLogger(LogConfig{LogFormat: LogFormatJson, LogLevel: Fatal}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: false, fatal: true}},
		{"json with error", NewLogger(LogConfig{LogFormat: LogFormatJson, LogLevel: "info"}).WithError(errors.New("my error")), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
		{"json changed field names", NewLogger(LogConfig{LogFormat: LogFormatJson, LogLevel: "info", LogFieldNames: LogFieldNames{
			Time: "time", Message: "message", Level: "lvl", Caller: "call", Error: "error",
		}}).WithError(errors.New("my error")), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
		{"logfmt debug", NewLogger(LogConfig{LogFormat: LogFormatLogfmt, LogLevel: Debug}), LevelEnabled{
			debug: true, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"logfmt info", NewLogger(LogConfig{LogFormat: LogFormatLogfmt, LogLevel: Info}), LevelEnabled{
			debug: false, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"logfmt warn", NewLogger(LogConfig{LogFormat: LogFormatLogfmt, LogLevel: Warn}), LevelEnabled{
			debug: false, info: false, warn: true, error: true, panic: true, fatal: true}},
		{"logfmt error", NewLogger(LogConfig{LogFormat: LogFormatLogfmt, LogLevel: Error}), LevelEnabled{
			debug: false, info: false, warn: false, error: true, panic: true, fatal: true}},
		{"logfmt panic", NewLogger(LogConfig{LogFormat: LogFormatLogfmt, LogLevel: Panic}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: true, fatal: true}},
		{"logfmt fatal", NewLogger(LogConfig{LogFormat: LogFormatLogfmt, LogLevel: Fatal}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: false, fatal: true}},
		{"logfmt with error", NewLogger(LogConfig{LogFormat: LogFormatLogfmt, LogLevel: "info"}).WithError(errors.New("my error")), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
		{"logfmt changed field names", NewLogger(LogConfig{LogFormat: LogFormatLogfmt, LogLevel: "info", LogFieldNames: LogFieldNames{
			Time: "time", Message: "message", Level: "lvl", Caller: "call", Error: "error",
		}}).WithError(errors.New("my error")), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
	}
	{
		for _, tc := range tt {
			tc.logger.Print("Print log ")
			tc.logger.Printf("Print log '%s'", tc.name)
			tc.logger.Debug("Debug log ")
			tc.logger.Debugf("Debug log '%s'", tc.name)
			tc.logger.Info("Info log ")
			tc.logger.Infof("Info log '%s'", tc.name)
			tc.logger.Warn("Warn log'")
			tc.logger.Warnf("Warn log '%s'", tc.name)
			tc.logger.Error("Error log")
			tc.logger.Errorf("Error log '%s'", tc.name)

			a.Equal(tc.levelEnabled.debug, tc.logger.IsDebug())
			a.Equal(tc.levelEnabled.info, tc.logger.IsInfo())
			a.Equal(tc.levelEnabled.warn, tc.logger.IsWarn())
			a.Equal(tc.levelEnabled.error, tc.logger.IsError())
			a.Equal(tc.levelEnabled.fatal, tc.logger.IsFatal())
			a.Equal(tc.levelEnabled.panic, tc.logger.IsPanic())
		}
	}
}

func TestContextLogger(t *testing.T) {
	parent := context.Background()
	c := context.WithValue(parent, ContextLogTag, Fields{
		"request_id": "4711",
	})

	log := GetInstance().WithContext(c)
	log.Info("Hello")

}
