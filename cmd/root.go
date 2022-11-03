package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/common/version"
	"go.uber.org/automaxprocs/maxprocs"
	"gopkg.in/alecthomas/kingpin.v2"
)

type setupFunc func(*run.Group, log.Logger, *prometheus.Registry) error

func Execute() {
	app := kingpin.New(filepath.Base(os.Args[0]), "MQTT Proxy")

	app.Version(version.Print("mqtt-proxy"))
	app.HelpFlag.Short('h')

	logConfig := log.LogConfig{}
	app.Flag("log.level", "Log filtering One of: [fatal, error, warn, info, debug]").Default(log.Info).EnumVar(&logConfig.LogLevel, log.Fatal, log.Error, log.Warn, log.Info, log.Debug)
	app.Flag("log.format", "Log format to use. One of: [logfmt, json, plain]").Default(log.LogFormatLogfmt).EnumVar(&logConfig.LogFormat, log.LogFormatLogfmt, log.LogFormatJson, log.LogFormatPlain)
	app.Flag("log.field-name.time", "Log time field name").Default(log.TimeKey).StringVar(&logConfig.LogFieldNames.Time)
	app.Flag("log.field-name.message", "Log message field name").Default(log.MessageKey).StringVar(&logConfig.LogFieldNames.Message)
	app.Flag("log.field-name.error", "Log error field name").Default(log.ErrorKey).StringVar(&logConfig.LogFieldNames.Error)
	app.Flag("log.field-name.caller", "Log caller field name").Default(log.CallerKey).StringVar(&logConfig.LogFieldNames.Caller)
	app.Flag("log.field-name.level", "Log time field name").Default(log.LevelKey).StringVar(&logConfig.LogFieldNames.Level)

	cmds := map[string]setupFunc{}

	registerServer(cmds, app)

	cmd, err := app.Parse(os.Args[1:])
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, fmt.Errorf("error parsing commandline arguments: %w", err))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}

	logger := log.NewLogger(logConfig)
	log.InitInstance(logger)

	undo, err := maxprocs.Set(maxprocs.Logger(func(template string, args ...interface{}) {
		logger.Debugf(template, args...)
	}))
	if undo != nil {
		defer undo()
	}
	if err != nil {
		logger.WithError(err).Infof("failed to set GOMAXPROCS")
	}

	metrics := prometheus.NewRegistry()
	metrics.MustRegister(
		version.NewCollector("mqtt_proxy"),
		collectors.NewGoCollector(
			collectors.WithGoCollectorRuntimeMetrics(),
		),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	var g run.Group

	if err := cmds[cmd](&g, logger, metrics); err != nil {
		// Use %+v for github.com/pkg/errors error to print with stack.
		logger.WithError(err).Fatalf("preparing %s command failed", cmd)
	}
	{
		cancel := make(chan struct{})
		g.Add(func() error {
			return interrupt(logger, cancel)
		}, func(error) {
			close(cancel)
		})
	}
	if err := g.Run(); err != nil {
		logger.WithError(err).Fatalf("%s command failed", cmd)
	}
	logger.Infof("exiting")
}

func interrupt(logger log.Logger, cancel <-chan struct{}) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-c:
		logger.WithField("signal", fmt.Sprintf("%s", s)).Infof("caught signal, exiting")
		return nil
	case <-cancel:
		return errors.New("canceled")
	}
}
