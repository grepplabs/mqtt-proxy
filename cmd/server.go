package cmd

import (
	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/grepplabs/mqtt-proxy/pkg/config"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	mqtthandler "github.com/grepplabs/mqtt-proxy/pkg/mqtt/handler"
	"github.com/grepplabs/mqtt-proxy/pkg/prober"
	"github.com/grepplabs/mqtt-proxy/pkg/publisher/instrument"
	"github.com/grepplabs/mqtt-proxy/pkg/publisher/kafka"
	"github.com/grepplabs/mqtt-proxy/pkg/publisher/noop"
	httpserver "github.com/grepplabs/mqtt-proxy/pkg/server/http"
	mqttserver "github.com/grepplabs/mqtt-proxy/pkg/server/mqtt"
	"github.com/grepplabs/mqtt-proxy/pkg/tls"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gopkg.in/alecthomas/kingpin.v2"
)

func registerServer(m map[string]setupFunc, app *kingpin.Application) {
	command := "server"

	cmd := app.Command(command, "mqtt-proxy server")

	cfg := new(config.Server)

	cmd.Flag("http.listen-address", "Listen host:port for HTTP endpoints.").Default("0.0.0.0:9090").StringVar(&cfg.HTTP.ListenAddress)
	cmd.Flag("http.grace-period", "Time to wait after an interrupt received for HTTP Server.").Default("10s").DurationVar(&cfg.HTTP.GracePeriod)

	cmd.Flag("mqtt.listen-address", "Listen host:port for MQTT endpoints.").Default("0.0.0.0:1883").StringVar(&cfg.MQTT.ListenAddress)
	cmd.Flag("mqtt.grace-period", "Time to wait after an interrupt received for MQTT Server.").Default("10s").DurationVar(&cfg.MQTT.GracePeriod)
	cmd.Flag("mqtt.read-timeout", "Maximum duration for reading the entire request.").Default("5s").DurationVar(&cfg.MQTT.ReadTimeout)
	cmd.Flag("mqtt.write-timeout", "Maximum duration before timing out writes of the response.").Default("5s").DurationVar(&cfg.MQTT.WriteTimeout)
	cmd.Flag("mqtt.idle-timeout", "Maximum duration before timing out writes of the response.").Default("0s").DurationVar(&cfg.MQTT.IdleTimeout)

	cmd.Flag("mqtt.reader-buffer-size", "Read buffer size pro tcp connection.").Default("1024").IntVar(&cfg.MQTT.ReaderBufferSize)
	cmd.Flag("mqtt.writer-buffer-size", "Write buffer size pro tcp connection.").Default("1024").IntVar(&cfg.MQTT.WriterBufferSize)

	cmd.Flag("mqtt.server-tls-cert", "TLS Certificate for MQTT server, leave blank to disable TLS").Default("").StringVar(&cfg.MQTT.TLSSrv.Cert)
	cmd.Flag("mqtt.server-tls-key", "TLS Key for the MQTT server, leave blank to disable TLS").Default("").StringVar(&cfg.MQTT.TLSSrv.Key)
	cmd.Flag("mqtt.server-tls-client-ca", "TLS CA to verify clients against. If no client CA is specified, there is no client verification on server side. (tls.NoClientCert)").Default("").StringVar(&cfg.MQTT.TLSSrv.ClientCA)

	cmd.Flag("mqtt.handler.ignore-unsupported", "List of unsupported messages which are ignored. One of: [SUBSCRIBE, UNSUBSCRIBE]").PlaceHolder("MSG").EnumsVar(&cfg.MQTT.Handler.IgnoreUnsupported, "SUBSCRIBE", "UNSUBSCRIBE")
	cmd.Flag("mqtt.handler.allow-unauthenticated", "List of messages for which connection is not disconnected if unauthenticated request is received. One of: [PUBLISH, PUBREL, PINGREQ]").PlaceHolder("MSG").EnumsVar(&cfg.MQTT.Handler.AllowUnauthenticated, "PUBLISH", "PUBREL", "PINGREQ")
	cmd.Flag("mqtt.handler.publish.timeout", "Maximum duration of sending publish request to broker.").Default("0s").DurationVar(&cfg.MQTT.Handler.Publish.Timeout)
	cmd.Flag("mqtt.handler.publish.async.at-most-once", "Async publish for AT_MOST_ONCE QoS.").Default("false").BoolVar(&cfg.MQTT.Handler.Publish.Async.AtMostOnce)
	cmd.Flag("mqtt.handler.publish.async.at-least-once", "Async publish for AT_LEAST_ONCE QoS.").Default("false").BoolVar(&cfg.MQTT.Handler.Publish.Async.AtLeastOnce)
	cmd.Flag("mqtt.handler.publish.async.exactly-once", "Async publish for EXACTLY_ONCE QoS.").Default("false").BoolVar(&cfg.MQTT.Handler.Publish.Async.ExactlyOnce)

	cmd.Flag("mqtt.publisher.name", "Publisher name. One of: [noop, kafka]").Default(config.Kafka).EnumVar(&cfg.MQTT.Publisher.Name, config.Noop, config.Kafka)
	cmd.Flag("mqtt.publisher.kafka.config", "Comma separated list of properties").PlaceHolder("PROP=VAL").SetValue(&cfg.MQTT.Publisher.Kafka.ConfArgs)
	cmd.Flag("mqtt.publisher.kafka.bootstrap-servers", "Kafka bootstrap servers").Default("localhost:9092").StringVar(&cfg.MQTT.Publisher.Kafka.BootstrapServers)
	cmd.Flag("mqtt.publisher.kafka.grace-period", "Time to wait after an interrupt received for Kafka publisher.").Default("10s").DurationVar(&cfg.MQTT.Publisher.Kafka.GracePeriod)
	cmd.Flag("mqtt.publisher.kafka.default-topic", "Default Kafka topic for MQTT publish messages").Default("").StringVar(&cfg.MQTT.Publisher.Kafka.DefaultTopic)
	cmd.Flag("mqtt.publisher.kafka.topic-mappings", "Comma separated list of Kafka topic to MQTT topic mappings").PlaceHolder("TOPIC=REGEX").SetValue(&cfg.MQTT.Publisher.Kafka.TopicMappings)

	m[command] = func(group *run.Group, logger log.Logger, registry *prometheus.Registry) error {
		return runServer(group, logger, registry, cfg)
	}
}

func runServer(
	group *run.Group,
	logger log.Logger,
	registry *prometheus.Registry,
	cfg *config.Server,
) error {

	httpProbe := prober.NewHTTP()
	{
		logger.Infof("setting up HTTP server")

		srv := httpserver.New(logger, registry, httpProbe,
			httpserver.WithListen(cfg.HTTP.ListenAddress),
			httpserver.WithGracePeriod(cfg.HTTP.GracePeriod),
		)
		group.Add(func() error {
			httpProbe.Healthy()
			return srv.ListenAndServe()
		}, func(err error) {
			httpProbe.NotReady(err)
			defer httpProbe.NotHealthy(err)

			srv.Shutdown(err)
		})
	}
	var publisher apis.Publisher
	{
		logger.Infof("setting publisher")

		var err error

		switch cfg.MQTT.Publisher.Name {
		case config.Noop:
			publisher, err = noop.New(logger, registry)
			if err != nil {
				return errors.Wrap(err, "setup noop publisher")
			}
		case config.Kafka:
			publisher, err = kafka.New(logger, registry,
				kafka.WithBootstrapServers(cfg.MQTT.Publisher.Kafka.BootstrapServers),
				kafka.WithDefaultTopic(cfg.MQTT.Publisher.Kafka.DefaultTopic),
				kafka.WithTopicMappings(cfg.MQTT.Publisher.Kafka.TopicMappings),
				kafka.WithConfigMap(cfg.MQTT.Publisher.Kafka.ConfArgs.ConfigMap()),
				kafka.WithGracePeriod(cfg.MQTT.Publisher.Kafka.GracePeriod),
			)
			if err != nil {
				return errors.Wrap(err, "setup kafka publisher")
			}
		default:
			return errors.Errorf("Unknown publisher %s", cfg.MQTT.Publisher.Name)
		}
		publisher = instrument.New(publisher, registry)

		group.Add(func() error {
			return publisher.Serve()
		}, func(err error) {
			publisher.Shutdown(err)
		})
	}
	{
		logger.Infof("setting up MQTT server")

		tlsCfg, err := tls.NewServerConfig(logger, cfg.MQTT.TLSSrv.Cert, cfg.MQTT.TLSSrv.Key, cfg.MQTT.TLSSrv.ClientCA)
		if err != nil {
			return errors.Wrap(err, "setup MQTT server")
		}

		handler := mqtthandler.New(logger, registry, publisher,
			mqtthandler.WithIgnoreUnsupported(cfg.MQTT.Handler.IgnoreUnsupported),
			mqtthandler.WithAllowUnauthenticated(cfg.MQTT.Handler.AllowUnauthenticated),
			mqtthandler.WithPublishTimeout(cfg.MQTT.Handler.Publish.Timeout),
			mqtthandler.WithPublishAsyncAtMostOnce(cfg.MQTT.Handler.Publish.Async.AtMostOnce),
			mqtthandler.WithPublishAsyncAtLeastOnce(cfg.MQTT.Handler.Publish.Async.AtLeastOnce),
			mqtthandler.WithPublishAsyncExactlyOnce(cfg.MQTT.Handler.Publish.Async.ExactlyOnce),
		)

		srv := mqttserver.New(logger, registry, httpProbe,
			mqttserver.WithListen(cfg.MQTT.ListenAddress),
			mqttserver.WithGracePeriod(cfg.MQTT.GracePeriod),
			mqttserver.WithReadTimeout(cfg.MQTT.ReadTimeout),
			mqttserver.WithWriteTimeout(cfg.MQTT.WriteTimeout),
			mqttserver.WithIdleTimeout(cfg.MQTT.IdleTimeout),
			mqttserver.WithReaderBufferSize(cfg.MQTT.ReaderBufferSize),
			mqttserver.WithWriterBufferSize(cfg.MQTT.WriterBufferSize),
			mqttserver.WithHandler(handler),
			mqttserver.WithTLSConfig(tlsCfg),
		)

		_ = promauto.With(registry).NewGaugeFunc(prometheus.GaugeOpts{
			Name: "mqtt_proxy_server_connections_active",
			Help: "Number of active TCP connections from clients to server.",
		}, func() float64 {
			return float64(srv.ActiveConnections())
		})

		_ = promauto.With(registry).NewCounterFunc(prometheus.CounterOpts{
			Name: "mqtt_proxy_server_connections_total",
			Help: "Total number of TCP connections from clients to server.",
		}, func() float64 {
			return float64(srv.TotalConnections())
		})

		group.Add(func() error {
			httpProbe.Ready()
			return srv.ListenAndServe()
		}, func(err error) {
			httpProbe.NotReady(err)

			srv.Shutdown(err)
		})
	}
	logger.Infof("starting MQTT server")
	return nil
}
