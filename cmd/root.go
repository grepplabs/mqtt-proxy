package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
	"go.uber.org/automaxprocs/maxprocs"
	"gopkg.in/alecthomas/kingpin.v2"
)

type setupFunc func(*run.Group, log.Logger, *prometheus.Registry) error

func Execute() {
	app := kingpin.New(filepath.Base(os.Args[0]), "MQTT Proxy")

	app.Version(version.Print("mqtt-proxy"))
	app.HelpFlag.Short('h')

	logLevel := app.Flag("log.level", "Log filtering One of: [fatal, error, warn, info, debug]").Default(log.Info).Enum(log.Fatal, log.Error, log.Warn, log.Info, log.Debug)
	logFormat := app.Flag("log.format", "Log format to use. One of: [logfmt, json, plain]").Default(log.LogFormatLogfmt).Enum(log.LogFormatLogfmt, log.LogFormatJson, log.LogFormatPlain)

	cmds := map[string]setupFunc{}

	registerServer(cmds, app)

	cmd, err := app.Parse(os.Args[1:])
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, errors.Wrapf(err, "error parsing commandline arguments"))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}

	logger := log.NewLogger(log.Configuration{
		LogLevel:  *logLevel,
		LogFormat: *logFormat,
	})

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
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
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
