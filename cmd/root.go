package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
	kongyaml "github.com/alecthomas/kong-yaml"
	"github.com/grepplabs/mqtt-proxy/pkg/config"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/common/version"
	"go.uber.org/automaxprocs/maxprocs"
)

type setupFunc func(*run.Group, log.Logger, *prometheus.Registry) error

type CLI struct {
	LogConfig log.Config    `embed:"" prefix:"log."`
	Server    config.Server `name:"server" cmd:"" help:"MQTT Proxy"`
	Version   struct{}      `name:"version" cmd:"" help:"Version information"`
}

func Execute() {
	cmds := map[string]setupFunc{}

	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name(os.Args[0]),
		kong.Description("MQTT Proxy"),
		kong.Configuration(kong.JSON, "/etc/mqtt-proxy/config.json", "~/.mqtt-proxy.json"),
		kong.Configuration(kongyaml.Loader, "/etc/mqtt-proxy/config.yaml", "~/.mqtt-proxy.yaml"),
		kong.UsageOnError(),
		log.Vars(), config.ServerVars())
	switch ctx.Command() {
	case "server":
		cmds[ctx.Command()] = func(group *run.Group, logger log.Logger, registry *prometheus.Registry) error {
			return runServer(group, logger, registry, &cli.Server)
		}
	case "version":
		fmt.Println(version.Print("mqtt-proxy"))
		os.Exit(0)
	default:
		fmt.Println(ctx.Command())
		os.Exit(1)
	}

	logger := log.NewLogger(cli.LogConfig)
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

	if err := cmds[ctx.Command()](&g, logger, metrics); err != nil {
		// Use %+v for github.com/pkg/errors error to print with stack.
		logger.WithError(err).Fatalf("preparing %s command failed", ctx.Command())
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
		logger.WithError(err).Fatalf("%s command failed", ctx.Command())
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
