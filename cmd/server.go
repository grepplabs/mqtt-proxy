package cmd

import (
	"crypto/tls"
	"fmt"
	"runtime"

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
	pubrabbitmq "github.com/grepplabs/mqtt-proxy/pkg/publisher/rabbitmq"
	pubsns "github.com/grepplabs/mqtt-proxy/pkg/publisher/sns"
	pubsqs "github.com/grepplabs/mqtt-proxy/pkg/publisher/sqs"
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
	logger.WithField("version", version.Version).WithField("branch", version.Branch).WithField("revision", version.Revision).Infof("starting mqtt-proxy on %s/%s", runtime.GOOS, runtime.GOARCH)

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
				pubkafka.WithMessageFormat(cfg.MQTT.Publisher.MessageFormat),
			)
			if err != nil {
				return fmt.Errorf("setup kafka publisher: %w", err)
			}
		case config.PublisherSQS:
			publisher, err = pubsqs.New(logger, registry,
				pubsqs.WithAWSProfile(cfg.MQTT.Publisher.SQS.AWSProfile),
				pubsqs.WithAWSRegion(cfg.MQTT.Publisher.SQS.AWSRegion),
				pubsqs.WithQueueMappings(cfg.MQTT.Publisher.SQS.QueueMappings),
				pubsqs.WithDefaultQueue(cfg.MQTT.Publisher.SQS.DefaultQueue),
				pubsqs.WithMessageFormat(cfg.MQTT.Publisher.MessageFormat),
			)
			if err != nil {
				return fmt.Errorf("setup sqs publisher: %w", err)
			}
		case config.PublisherSNS:
			publisher, err = pubsns.New(logger, registry,
				pubsns.WithAWSProfile(cfg.MQTT.Publisher.SNS.AWSProfile),
				pubsns.WithAWSRegion(cfg.MQTT.Publisher.SNS.AWSRegion),
				pubsns.WithTopicARNMappings(cfg.MQTT.Publisher.SNS.TopicARNMappings),
				pubsns.WithDefaultTopicARN(cfg.MQTT.Publisher.SNS.DefaultTopicARN),
				pubsns.WithMessageFormat(cfg.MQTT.Publisher.MessageFormat),
			)
			if err != nil {
				return fmt.Errorf("setup sns publisher: %w", err)
			}
		case config.PublisherRabbitMQ:
			publisher, err = pubrabbitmq.New(logger, registry,
				pubrabbitmq.WithScheme(cfg.MQTT.Publisher.RabbitMQ.Scheme),
				pubrabbitmq.WithHost(cfg.MQTT.Publisher.RabbitMQ.Host),
				pubrabbitmq.WithPort(cfg.MQTT.Publisher.RabbitMQ.Port),
				pubrabbitmq.WithUsername(cfg.MQTT.Publisher.RabbitMQ.Username),
				pubrabbitmq.WithPassword(cfg.MQTT.Publisher.RabbitMQ.Password),
				pubrabbitmq.WithVHost(cfg.MQTT.Publisher.RabbitMQ.VHost),
				pubrabbitmq.WithExchange(cfg.MQTT.Publisher.RabbitMQ.Exchange),
				pubrabbitmq.WithConnectionTimeout(cfg.MQTT.Publisher.RabbitMQ.ConnectionTimeout),
				pubrabbitmq.WithRequestTimeout(cfg.MQTT.Publisher.RabbitMQ.RequestTimeout),
				pubrabbitmq.WithQueueMappings(cfg.MQTT.Publisher.RabbitMQ.QueueMappings),
				pubrabbitmq.WithDefaultQueue(cfg.MQTT.Publisher.RabbitMQ.DefaultQueue),
				pubrabbitmq.WithMessageFormat(cfg.MQTT.Publisher.MessageFormat),
				pubrabbitmq.WithPublisherConfirmsAtLeastOnce(cfg.MQTT.Publisher.RabbitMQ.PublisherConfirms.AtLeastOnce),
				pubrabbitmq.WithPublisherConfirmsAtMostOnce(cfg.MQTT.Publisher.RabbitMQ.PublisherConfirms.AtMostOnce),
				pubrabbitmq.WithPublisherConfirmsExactlyOnce(cfg.MQTT.Publisher.RabbitMQ.PublisherConfirms.ExactlyOnce),
			)
			if err != nil {
				return fmt.Errorf("setup rabbitmq publisher: %w", err)
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
