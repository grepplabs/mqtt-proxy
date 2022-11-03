package cmd

import (
	"crypto/tls"
	"fmt"

	"github.com/grepplabs/mqtt-proxy/apis"
	authinst "github.com/grepplabs/mqtt-proxy/pkg/auth/instrument"
	authnoop "github.com/grepplabs/mqtt-proxy/pkg/auth/noop"
	authplain "github.com/grepplabs/mqtt-proxy/pkg/auth/plain"
	"github.com/grepplabs/mqtt-proxy/pkg/config"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	mqtthandler "github.com/grepplabs/mqtt-proxy/pkg/mqtt/handler"
	"github.com/grepplabs/mqtt-proxy/pkg/prober"
	pubinst "github.com/grepplabs/mqtt-proxy/pkg/publisher/instrument"
	pubkafka "github.com/grepplabs/mqtt-proxy/pkg/publisher/kafka"
	pubnoop "github.com/grepplabs/mqtt-proxy/pkg/publisher/noop"
	httpserver "github.com/grepplabs/mqtt-proxy/pkg/server/http"
	mqttserver "github.com/grepplabs/mqtt-proxy/pkg/server/mqtt"
	servertls "github.com/grepplabs/mqtt-proxy/pkg/tls"
	"github.com/grepplabs/mqtt-proxy/pkg/tls/cert/filesource"
	tlscert "github.com/grepplabs/mqtt-proxy/pkg/tls/cert/source"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gopkg.in/alecthomas/kingpin.v2"
)

func registerServer(m map[string]setupFunc, app *kingpin.Application) {
	command := "server"

	cmd := app.Command(command, "mqtt-proxy server")

	cfg := new(config.Server)
	cfg.Init()

	cmd.Flag("http.listen-address", "Listen host:port for HTTP endpoints.").Default("0.0.0.0:9090").StringVar(&cfg.HTTP.ListenAddress)
	cmd.Flag("http.grace-period", "Time to wait after an interrupt received for HTTP Server.").Default("10s").DurationVar(&cfg.HTTP.GracePeriod)

	cmd.Flag("mqtt.listen-address", "Listen host:port for MQTT endpoints.").Default("0.0.0.0:1883").StringVar(&cfg.MQTT.ListenAddress)
	cmd.Flag("mqtt.grace-period", "Time to wait after an interrupt received for MQTT Server.").Default("10s").DurationVar(&cfg.MQTT.GracePeriod)
	cmd.Flag("mqtt.read-timeout", "Maximum duration for reading the entire request.").Default("5s").DurationVar(&cfg.MQTT.ReadTimeout)
	cmd.Flag("mqtt.write-timeout", "Maximum duration before timing out writes of the response.").Default("5s").DurationVar(&cfg.MQTT.WriteTimeout)
	cmd.Flag("mqtt.idle-timeout", "Maximum duration before timing out writes of the response.").Default("0s").DurationVar(&cfg.MQTT.IdleTimeout)

	cmd.Flag("mqtt.reader-buffer-size", "Read buffer size pro tcp connection.").Default("1024").IntVar(&cfg.MQTT.ReaderBufferSize)
	cmd.Flag("mqtt.writer-buffer-size", "Write buffer size pro tcp connection.").Default("1024").IntVar(&cfg.MQTT.WriterBufferSize)

	cmd.Flag("mqtt.server-tls.enable", "Enable server side TLS").Default("false").BoolVar(&cfg.MQTT.TLSSrv.Enable)
	cmd.Flag("mqtt.server-tls.cert-source", "TLS certificate source").Default(config.CertSourceFile).EnumVar(&cfg.MQTT.TLSSrv.CertSource, config.CertSourceFile)
	cmd.Flag("mqtt.server-tls.refresh", "Option to specify the refresh interval for the TLS certificates.").Default("0s").DurationVar(&cfg.MQTT.TLSSrv.Refresh)

	cmd.Flag("mqtt.server-tls.file.cert", "TLS Certificate for MQTT server, leave blank to disable TLS").Default("").StringVar(&cfg.MQTT.TLSSrv.File.Cert)
	cmd.Flag("mqtt.server-tls.file.key", "TLS Key for the MQTT server, leave blank to disable TLS").Default("").StringVar(&cfg.MQTT.TLSSrv.File.Key)
	cmd.Flag("mqtt.server-tls.file.client-ca", "TLS CA to verify clients against. If no client CA is specified, there is no client verification on server side.").Default("").StringVar(&cfg.MQTT.TLSSrv.File.ClientCA)
	cmd.Flag("mqtt.server-tls.file.client-clr", "TLS X509 CLR signed be the client CA. If no revocation list is specified, only client CA is verified").Default("").StringVar(&cfg.MQTT.TLSSrv.File.ClientCLR)

	cmd.Flag("mqtt.handler.ignore-unsupported", "List of unsupported messages which are ignored. One of: [SUBSCRIBE, UNSUBSCRIBE]").PlaceHolder("MSG").EnumsVar(&cfg.MQTT.Handler.IgnoreUnsupported, "SUBSCRIBE", "UNSUBSCRIBE")
	cmd.Flag("mqtt.handler.allow-unauthenticated", "List of messages for which connection is not disconnected if unauthenticated request is received. One of: [PUBLISH, PUBREL, PINGREQ]").PlaceHolder("MSG").EnumsVar(&cfg.MQTT.Handler.AllowUnauthenticated, "PUBLISH", "PUBREL", "PINGREQ")
	cmd.Flag("mqtt.handler.publish.timeout", "Maximum duration of sending publish request to broker.").Default("0s").DurationVar(&cfg.MQTT.Handler.Publish.Timeout)
	cmd.Flag("mqtt.handler.publish.async.at-most-once", "Async publish for AT_MOST_ONCE QoS.").Default("false").BoolVar(&cfg.MQTT.Handler.Publish.Async.AtMostOnce)
	cmd.Flag("mqtt.handler.publish.async.at-least-once", "Async publish for AT_LEAST_ONCE QoS.").Default("false").BoolVar(&cfg.MQTT.Handler.Publish.Async.AtLeastOnce)
	cmd.Flag("mqtt.handler.publish.async.exactly-once", "Async publish for EXACTLY_ONCE QoS.").Default("false").BoolVar(&cfg.MQTT.Handler.Publish.Async.ExactlyOnce)

	cmd.Flag("mqtt.handler.auth.name", "Authenticator name. One of: [noop, plain]").Default(config.AuthNoop).EnumVar(&cfg.MQTT.Handler.Authenticator.Name, config.AuthNoop, config.AuthPlain)
	cmd.Flag("mqtt.handler.auth.plain.credentials", "List of username and password fields.").Default("USERNAME=PASSWORD").StringMapVar(&cfg.MQTT.Handler.Authenticator.Plain.Credentials)
	cmd.Flag("mqtt.handler.auth.plain.credentials-file", "Location of a headerless CSV file containing `usernanme,password` records").Default("").StringVar(&cfg.MQTT.Handler.Authenticator.Plain.CredentialsFile)

	cmd.Flag("mqtt.publisher.name", "Publisher name. One of: [noop, kafka]").Default(config.PublisherNoop).EnumVar(&cfg.MQTT.Publisher.Name, config.PublisherNoop, config.PublisherKafka)
	cmd.Flag("mqtt.publisher.kafka.config", "Comma separated list of properties").PlaceHolder("PROP=VAL").SetValue(&cfg.MQTT.Publisher.Kafka.ConfArgs)
	cmd.Flag("mqtt.publisher.kafka.bootstrap-servers", "Kafka bootstrap servers").Default("localhost:9092").StringVar(&cfg.MQTT.Publisher.Kafka.BootstrapServers)
	cmd.Flag("mqtt.publisher.kafka.grace-period", "Time to wait after an interrupt received for Kafka publisher.").Default("10s").DurationVar(&cfg.MQTT.Publisher.Kafka.GracePeriod)
	cmd.Flag("mqtt.publisher.kafka.default-topic", "Default Kafka topic for MQTT publish messages").Default("").StringVar(&cfg.MQTT.Publisher.Kafka.DefaultTopic)
	cmd.Flag("mqtt.publisher.kafka.topic-mappings", "Comma separated list of Kafka topic to MQTT topic mappings").PlaceHolder("TOPIC=REGEX").SetValue(&cfg.MQTT.Publisher.Kafka.TopicMappings)
	cmd.Flag("mqtt.publisher.kafka.workers", "Number of kafka publisher workers").Default("1").IntVar(&cfg.MQTT.Publisher.Kafka.Workers)

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

	err := cfg.Validate()
	if err != nil {
		return err
	}

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

	var authenticator apis.UserPasswordAuthenticator
	{
		logger.Infof("setting up authenticator %s", cfg.MQTT.Handler.Authenticator.Name)

		switch cfg.MQTT.Handler.Authenticator.Name {
		case config.AuthNoop:
			authenticator = authnoop.New(logger, registry)
		case config.AuthPlain:
			authenticator, err = authplain.New(logger, registry,
				authplain.WithCredentials(cfg.MQTT.Handler.Authenticator.Plain.Credentials),
				authplain.WithCredentialsFile(cfg.MQTT.Handler.Authenticator.Plain.CredentialsFile),
			)
			if err != nil {
				return fmt.Errorf("setup plain authenticator: %w", err)
			}
		default:
			return fmt.Errorf("unknown authenticator %s", cfg.MQTT.Handler.Authenticator.Name)
		}
		authenticator = authinst.New(authenticator, registry)
		defer func() {
			err := authenticator.Close()
			if err != nil {
				logger.WithError(err).Warnf("authenticator close failed")
			}
		}()
	}
	var publisher apis.Publisher
	{
		logger.Infof("setting up publisher %s", cfg.MQTT.Publisher.Name)

		var err error

		switch cfg.MQTT.Publisher.Name {
		case config.PublisherNoop:
			publisher = pubnoop.New(logger, registry)
		case config.PublisherKafka:
			publisher, err = pubkafka.New(logger, registry,
				pubkafka.WithBootstrapServers(cfg.MQTT.Publisher.Kafka.BootstrapServers),
				pubkafka.WithDefaultTopic(cfg.MQTT.Publisher.Kafka.DefaultTopic),
				pubkafka.WithTopicMappings(cfg.MQTT.Publisher.Kafka.TopicMappings),
				pubkafka.WithConfigMap(cfg.MQTT.Publisher.Kafka.ConfArgs.ConfigMap()),
				pubkafka.WithGracePeriod(cfg.MQTT.Publisher.Kafka.GracePeriod),
				pubkafka.WithWorkers(cfg.MQTT.Publisher.Kafka.Workers),
			)
			if err != nil {
				return fmt.Errorf("setup kafka publisher: %w", err)
			}
		default:
			return fmt.Errorf("unknown publisher %s", cfg.MQTT.Publisher.Name)
		}
		publisher = pubinst.New(publisher, registry)

		group.Add(func() error {
			return publisher.Serve()
		}, func(err error) {
			publisher.Shutdown(err)
		})
	}
	{
		logger.Infof("setting up MQTT server")

		var tlsConfig *tls.Config
		if cfg.MQTT.TLSSrv.Enable {
			logger.Infof("enabling server side TLS")
			var (
				source tlscert.ServerSource
				err    error
			)
			switch cfg.MQTT.TLSSrv.CertSource {
			case config.CertSourceFile:
				source, err = filesource.New(
					filesource.WithLogger(logger),
					filesource.WithX509KeyPair(cfg.MQTT.TLSSrv.File.Cert, cfg.MQTT.TLSSrv.File.Key),
					filesource.WithClientAuthFile(cfg.MQTT.TLSSrv.File.ClientCA),
					filesource.WithClientCRLFile(cfg.MQTT.TLSSrv.File.ClientCLR),
					filesource.WithRefresh(cfg.MQTT.TLSSrv.Refresh),
				)
				if err != nil {
					return fmt.Errorf("setup cert file source: %w", err)
				}
			default:
				return fmt.Errorf("unknown cert source %s", cfg.MQTT.TLSSrv.CertSource)
			}
			tlsConfig, err = servertls.NewServerConfig(logger, source)
			if err != nil {
				return fmt.Errorf("setup server TLS config: %w", err)
			}
		}

		handler := mqtthandler.New(logger, registry, publisher,
			mqtthandler.WithIgnoreUnsupported(cfg.MQTT.Handler.IgnoreUnsupported),
			mqtthandler.WithAllowUnauthenticated(cfg.MQTT.Handler.AllowUnauthenticated),
			mqtthandler.WithPublishTimeout(cfg.MQTT.Handler.Publish.Timeout),
			mqtthandler.WithPublishAsyncAtMostOnce(cfg.MQTT.Handler.Publish.Async.AtMostOnce),
			mqtthandler.WithPublishAsyncAtLeastOnce(cfg.MQTT.Handler.Publish.Async.AtLeastOnce),
			mqtthandler.WithPublishAsyncExactlyOnce(cfg.MQTT.Handler.Publish.Async.ExactlyOnce),
			mqtthandler.WithAuthenticator(authenticator),
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
			mqttserver.WithTLSConfig(tlsConfig),
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
