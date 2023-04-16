package rabbitmq

import (
	"context"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestChannel(t *testing.T) {
	if os.Getenv("IT") != "yes" {
		t.Skip("Skipping IT test")
	}
	url := "amqp://user:bitnami@localhost:5672/"
	payload := &Payload{
		Body:            []byte("test"),
		ContentType:     "text/plain",
		ContentEncoding: "",
	}
	tests := []struct {
		name        string
		channelFunc func() (Channel, error)
	}{
		{
			name: "channel",
			channelFunc: func() (Channel, error) {
				return NewChannel(url)
			},
		},
		{
			name: "channel confirm",
			channelFunc: func() (Channel, error) {
				return NewChannel(url, WithChannelPublisherConfirms())
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			channel, err := tc.channelFunc()
			require.Nil(t, err)
			defer channel.Close()
			_, err = channel.Publish(ctx, "", "test", payload, nil)
			require.Nil(t, err)
		})
	}
}
