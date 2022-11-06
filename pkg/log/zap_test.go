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
		{"plain debug", NewLogger(Config{Format: FormatPlain, Level: "debug"}), LevelEnabled{
			debug: true, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"plain info", NewLogger(Config{Format: FormatPlain, Level: "info"}), LevelEnabled{
			debug: false, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"plain warn", NewLogger(Config{Format: FormatPlain, Level: "warn"}), LevelEnabled{
			debug: false, info: false, warn: true, error: true, panic: true, fatal: true}},
		{"plain error", NewLogger(Config{Format: FormatPlain, Level: "error"}), LevelEnabled{
			debug: false, info: false, warn: false, error: true, panic: true, fatal: true}},
		{"plain panic", NewLogger(Config{Format: FormatPlain, Level: "panic"}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: true, fatal: true}},
		{"plain fatal", NewLogger(Config{Format: FormatPlain, Level: "fatal"}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: false, fatal: true}},
		{"plain with error", NewLogger(Config{Format: FormatPlain, Level: "info"}).WithError(errors.New("my error")), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
		{"json debug", NewLogger(Config{Format: FormatJson, Level: Debug}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: true, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"json info", NewLogger(Config{Format: FormatJson, Level: Info}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: false, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"json warn", NewLogger(Config{Format: FormatJson, Level: Warn}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: false, info: false, warn: true, error: true, panic: true, fatal: true}},
		{"json error", NewLogger(Config{Format: FormatJson, Level: Error}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: false, info: false, warn: false, error: true, panic: true, fatal: true}},
		{"json panic", NewLogger(Config{Format: FormatJson, Level: Panic}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: true, fatal: true}},
		{"json fatal", NewLogger(Config{Format: FormatJson, Level: Fatal}).WithFields(Fields{"tag": "value"}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: false, fatal: true}},
		{"json with error", NewLogger(Config{Format: FormatJson, Level: "info"}).WithError(errors.New("my error")), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
		{"json changed field names", NewLogger(Config{Format: FormatJson, Level: "info", FieldNames: FieldNames{
			Time: "time", Message: "message", Level: "lvl", Caller: "call", Error: "error",
		}}).WithError(errors.New("my error")), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
		{"logfmt debug", NewLogger(Config{Format: FormatLogfmt, Level: Debug}), LevelEnabled{
			debug: true, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"logfmt info", NewLogger(Config{Format: FormatLogfmt, Level: Info}), LevelEnabled{
			debug: false, info: true, warn: true, error: true, panic: true, fatal: true}},
		{"logfmt warn", NewLogger(Config{Format: FormatLogfmt, Level: Warn}), LevelEnabled{
			debug: false, info: false, warn: true, error: true, panic: true, fatal: true}},
		{"logfmt error", NewLogger(Config{Format: FormatLogfmt, Level: Error}), LevelEnabled{
			debug: false, info: false, warn: false, error: true, panic: true, fatal: true}},
		{"logfmt panic", NewLogger(Config{Format: FormatLogfmt, Level: Panic}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: true, fatal: true}},
		{"logfmt fatal", NewLogger(Config{Format: FormatLogfmt, Level: Fatal}), LevelEnabled{
			debug: false, info: false, warn: false, error: false, panic: false, fatal: true}},
		{"logfmt with error", NewLogger(Config{Format: FormatLogfmt, Level: "info"}).WithError(errors.New("my error")), LevelEnabled{
			debug: false, info: true, warn: true, error: true, fatal: true, panic: true}},
		{"logfmt changed field names", NewLogger(Config{Format: FormatLogfmt, Level: "info", FieldNames: FieldNames{
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
