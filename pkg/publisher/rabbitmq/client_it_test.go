package rabbitmq

import (
	"context"
	"github.com/grepplabs/mqtt-proxy/pkg/config"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestClient(t *testing.T) {
	if os.Getenv("IT") != "yes" {
		t.Skip("Skipping IT test")
	}
	logger := log.NewLogger(log.Config{
		Level:  log.Debug,
		Format: log.FormatLogfmt,
	})
	options := options{
		scheme:            "amqp",
		host:              "localhost",
		port:              5672,
		username:          "user",
		password:          "bitnami",
		vhost:             "/",
		connectionTimeout: 0,
		requestTimeout:    0,
		exchange:          "",
		defaultQueue:      "test",
		queueMappings:     config.TopicMappings{},
		messageFormat:     "",
	}
	client, err := NewClient(logger, false, options)
	require.Nil(t, err)

	payload := &Payload{
		Body:            []byte("test"),
		ContentType:     "text/plain",
		ContentEncoding: "",
	}
	_, err = client.Publish(context.Background(), options.exchange, options.defaultQueue, payload, nil)
	require.Nil(t, err)
}
