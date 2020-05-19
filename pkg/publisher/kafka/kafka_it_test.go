package kafka

import (
	"context"
	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/oklog/run"
	"os"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
)

func newTestPublisherOrExit() *Publisher {
	logger := log.NewLogger(log.Configuration{
		LogLevel:  log.Debug,
		LogFormat: log.LogFormatLogfmt,
	})

	var registry *prometheus.Registry

	defaultConfig := make(kafka.ConfigMap)
	_ = defaultConfig.SetKey("producer.message.timeout.ms", 5000)

	bootstrapServers := "172.17.0.1:19092"

	publisher, err := New(logger, registry,
		WithBootstrapServers(bootstrapServers),
		WithGracePeriod(60*time.Second),
		WithConfigMap(defaultConfig))

	if err != nil {
		logger.WithError(err).Errorf("kafka publisher creation failed")
		os.Exit(1)
	}
	return publisher
}

func TestPublishIT(t *testing.T) {
	/*
		if os.Getenv("IT") != "yes" {
			t.Skip("Skipping IT test")
		}
	*/

	publisher := newTestPublisherOrExit()
	logger := publisher.logger

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		publisher.Serve()
	}()

	go func() {
		<-ctx.Done()
		publisher.Shutdown(nil)
	}()

	req := &apis.PublishRequest{
		TopicName: "temperature",
		Message:   []byte("hot"),
	}
	res, err := publisher.Publish(ctx, req)
	if err != nil {
		logger.WithError(err).Errorf("kafka publish error")
	} else {
		logger.Printf("##### publish response: ID=%v, Error=%v", res.ID, res.Error)
	}

	logger.Printf("Finished")
}

func TestPublishAsyncIT(t *testing.T) {
	publisher := newTestPublisherOrExit()

	logger := publisher.logger

	var group run.Group

	group.Add(func() error {
		req := &apis.PublishRequest{
			TopicName: "temperature",
			Message:   []byte("hot"),
			Qos:       1,
		}
		logger.Printf("---> publish request")

		err := publisher.PublishAsync(context.Background(), req, func(request *apis.PublishRequest, res *apis.PublishResponse) {
			logger.Printf("<--- publish response: ID=%v, Error=%v", res.ID, res.Error)
		})
		if err != nil {
			logger.WithError(err).Errorf("kafka publish async error")
			return err
		}
		logger.Infof("Sleep after publish ..... ")
		time.Sleep(10 * time.Second)

		return nil
	}, func(err error) {
	})

	group.Add(func() error {
		logger.Infof("publisher serving")
		return publisher.Serve()
	}, func(err error) {
		logger.Printf("### shutdown")
		publisher.Shutdown(err)
	})

	err := group.Run()
	if err != nil {
		logger.WithError(err).Fatalf("Run error")
	}
	logger.Printf("Finished")
}

func TestPublisherFactory(t *testing.T) {
	var apiPublisher apis.Publisher
	apiPublisher = newTestPublisherOrExit()
	_ = apiPublisher
}
