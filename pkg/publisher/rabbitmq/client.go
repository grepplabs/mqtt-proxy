package rabbitmq

import (
	"context"
	"fmt"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
)

type Payload struct {
	Body            []byte
	ContentType     string
	ContentEncoding string
}

type Client interface {
	Publish(ctx context.Context, exchange, routingKey string, payload *Payload, headers map[string]any) (uint64, error)
	Close() error
}

type client struct {
	logger          log.Logger
	options         options
	channelProvider *ChannelProvider
}

func NewClient(logger log.Logger, publisherConfirms bool, options options) (Client, error) {
	uri := amqpURI(options)
	channelOpts := make([]ChannelOptionFunc, 0)
	if publisherConfirms {
		channelOpts = append(channelOpts, WithChannelPublisherConfirms())
	}
	channelProvider := NewChannelProvider(uri, channelOpts...)
	return &client{
		logger:          logger,
		options:         options,
		channelProvider: channelProvider,
	}, nil
}

func (c *client) Publish(ctx context.Context, exchange, routingKey string, payload *Payload, headers map[string]any) (uint64, error) {
	deliveryTag, err := c.doPublish(ctx, exchange, routingKey, payload, headers)
	if err != nil {
		if shouldRetry(err) {
			return c.doPublish(ctx, exchange, routingKey, payload, headers)
		}
		return 0, err
	}
	return deliveryTag, nil
}

func (c *client) doPublish(ctx context.Context, exchange, routingKey string, payload *Payload, headers map[string]any) (uint64, error) {
	if int64(c.options.requestTimeout) > 0 {
		var cancelFunc context.CancelFunc
		ctx, cancelFunc = context.WithTimeout(ctx, c.options.requestTimeout)
		defer cancelFunc()
	}
	ch, err := c.channelProvider.GetChannel()
	if err != nil {
		return 0, err
	}
	deliveryTag, err := ch.Publish(ctx, exchange, routingKey, payload, headers)
	if err != nil {
		c.channelProvider.CloseChannel(ch)
		return 0, err
	}
	return deliveryTag, nil
}

func (c *client) Close() error {
	return c.channelProvider.Close()
}

func amqpURI(opts options) string {
	if opts.username != "" {
		return fmt.Sprintf("%s://%s:%s@%s:%d/%s", opts.scheme, opts.username, opts.password, opts.host, opts.port, opts.vhost)
	}
	return fmt.Sprintf("%s://%s:%d/%s", opts.scheme, opts.host, opts.port, opts.vhost)
}
