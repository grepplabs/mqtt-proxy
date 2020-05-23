package log

import (
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
		{"plain debug", NewLogger(Configuration{LogFormat: LogFormatPlain, LogLevel: "debug"}), LevelEnabled{
			debug: true, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"plain info", NewLogger(Configuration{LogFormat: LogFormatPlain, LogLevel: "info"}), LevelEnabled{
			debug: false, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"plain warn", NewLogger(Configuration{LogFormat: LogFormatPlain, LogLevel: "warn"}), LevelEnabled{
			debug: false, info: false, warn: true, error: true, panic: true, fatal: true}},
		{"plain error", NewLogger(Configuration{LogFormat: LogFormatPlain, LogLevel: "error"}), LevelEnabled{
			debug: false, info: false, warn: false, error: true, panic: true, fatal: true}},
		{"plain panic", NewLogger(Configuration{LogFormat: LogFormatPlain, LogLevel: "panic"}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: true, fatal: true}},
		{"plain fatal", NewLogger(Configuration{LogFormat: LogFormatPlain, LogLevel: "fatal"}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: false, fatal: true}},
		{"plain with error", NewLogger(Configuration{LogFormat: LogFormatPlain, LogLevel: "info"}).WithError(errors.New("my error")), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
		{"json debug", NewLogger(Configuration{LogFormat: LogFormatJson, LogLevel: Debug}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: true, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"json info", NewLogger(Configuration{LogFormat: LogFormatJson, LogLevel: Info}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: false, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"json warn", NewLogger(Configuration{LogFormat: LogFormatJson, LogLevel: Warn}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: false, info: false, warn: true, error: true, panic: true, fatal: true}},
		{"json error", NewLogger(Configuration{LogFormat: LogFormatJson, LogLevel: Error}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: false, info: false, warn: false, error: true, panic: true, fatal: true}},
		{"json panic", NewLogger(Configuration{LogFormat: LogFormatJson, LogLevel: Panic}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: true, fatal: true}},
		{"json fatal", NewLogger(Configuration{LogFormat: LogFormatJson, LogLevel: Fatal}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: false, fatal: true}},
		{"json with error", NewLogger(Configuration{LogFormat: LogFormatJson, LogLevel: "info"}).WithError(errors.New("my error")), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
		{"json changed field names", NewLogger(Configuration{LogFormat: LogFormatJson, LogLevel: "info", LogFieldNames: LogFieldNames{
			Time: "time", Message: "message", Level: "lvl", Caller: "call", Error: "error",
		}}).WithError(errors.New("my error")), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
		{"logfmt debug", NewLogger(Configuration{LogFormat: LogFormatLogfmt, LogLevel: Debug}), LevelEnabled{
			debug: true, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"logfmt info", NewLogger(Configuration{LogFormat: LogFormatLogfmt, LogLevel: Info}), LevelEnabled{
			debug: false, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"logfmt warn", NewLogger(Configuration{LogFormat: LogFormatLogfmt, LogLevel: Warn}), LevelEnabled{
			debug: false, info: false, warn: true, error: true, panic: true, fatal: true}},
		{"logfmt error", NewLogger(Configuration{LogFormat: LogFormatLogfmt, LogLevel: Error}), LevelEnabled{
			debug: false, info: false, warn: false, error: true, panic: true, fatal: true}},
		{"logfmt panic", NewLogger(Configuration{LogFormat: LogFormatLogfmt, LogLevel: Panic}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: true, fatal: true}},
		{"logfmt fatal", NewLogger(Configuration{LogFormat: LogFormatLogfmt, LogLevel: Fatal}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: false, fatal: true}},
		{"logfmt with error", NewLogger(Configuration{LogFormat: LogFormatLogfmt, LogLevel: "info"}).WithError(errors.New("my error")), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
		{"logfmt changed field names", NewLogger(Configuration{LogFormat: LogFormatLogfmt, LogLevel: "info", LogFieldNames: LogFieldNames{
			Time: "time", Message: "message", Level: "lvl", Caller: "call", Error: "error",
		}}).WithError(errors.New("my error")), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
	}
	{
		for _, tc := range tt {
			tc.logger.Debugf("Debug log '%s'", tc.name)
			tc.logger.Infof("Info log '%s'", tc.name)
			tc.logger.Warnf("Warn log '%s'", tc.name)
			tc.logger.Errorf("WithError log '%s'", tc.name)

			a.Equal(tc.levelEnabled.debug, tc.logger.IsDebug())
			a.Equal(tc.levelEnabled.info, tc.logger.IsInfo())
			a.Equal(tc.levelEnabled.warn, tc.logger.IsWarn())
			a.Equal(tc.levelEnabled.error, tc.logger.IsError())
			a.Equal(tc.levelEnabled.fatal, tc.logger.IsFatal())
			a.Equal(tc.levelEnabled.panic, tc.logger.IsPanic())
		}
	}
}
