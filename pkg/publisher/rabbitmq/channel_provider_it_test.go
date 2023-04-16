package rabbitmq

import (
	"context"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestChannelProvider(t *testing.T) {
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
		name                string
		channelProviderFunc func() *ChannelProvider
	}{
		{
			name: "channel",
			channelProviderFunc: func() *ChannelProvider {
				return NewChannelProvider(url)
			},
		},
		{
			name: "channel confirm",
			channelProviderFunc: func() *ChannelProvider {
				return NewChannelProvider(url, WithChannelPublisherConfirms())
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			provider := tc.channelProviderFunc()
			defer provider.Close()

			for i := 0; i < 2; i++ {
				channel, err := provider.GetChannel()
				require.Nil(t, err)
				_, err = channel.Publish(ctx, "", "test", payload, nil)
				require.Nil(t, err)
			}
		})
	}
}
