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
	"github.com/prometheus/common/version"
)

func runServer(
	group *run.Group,
	logger log.Logger,
	registry *prometheus.Registry,
	cfg *config.Server,
) error {
	logger.WithField("version", version.Version).WithField("branch", version.Branch).WithField("revision", version.Revision).Infof("starting mqtt-proxy")

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
