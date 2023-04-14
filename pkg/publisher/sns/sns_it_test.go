package sqs

import (
	"context"
	"fmt"
	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/grepplabs/mqtt-proxy/pkg/config"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/oklog/run"
	"os"
	"testing"
)

func newTestPublisherOrExit() *Publisher {
	publisher, err := New(
		log.NewLogger(log.Config{
			Level:  log.Debug,
			Format: log.FormatLogfmt,
		}), nil,
		WithDefaultTopicARN("arn:aws:sns:eu-central-1:123456789012:test1"),
		WithTopicARNMappings(config.TopicMappings{}),
		WithAWSProfile("admin-dev"),
		WithAWSRegion("eu-central-1"),
		WithMessageFormat(config.MessageFormatPlain),
	)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "sqs publisher creation failed: %v", err)
		os.Exit(1)
	}
	return publisher
}

func TestPublishIT(t *testing.T) {
	if os.Getenv("IT") != "yes" {
		t.Skip("Skipping IT test")
	}

	publisher := newTestPublisherOrExit()
	logger := publisher.logger

	logger.Infof("Publisher created")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		logger.Infof("%v", publisher.Serve())
	}()

	go func() {
		<-ctx.Done()
		publisher.Shutdown(nil)
	}()

	req := &apis.PublishRequest{
		TopicName: "temperature",
		Message:   []byte("hot"),
	}

	for i := 0; i < 2; i++ {
		res, err := publisher.Publish(ctx, req)
		if err != nil {
			logger.WithError(err).Errorf("sqs publish error")
		} else {
			logger.Printf("##### publish response: ID=%v, Error=%v", res.ID, res.Error)
		}
	}

	logger.Printf("Finished")
}

func TestPublishAsyncIT(t *testing.T) {
	if os.Getenv("IT") != "yes" {
		t.Skip("Skipping IT test")
	}
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
