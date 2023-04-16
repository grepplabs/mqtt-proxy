package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"github.com/grepplabs/mqtt-proxy/pkg/util"
	"github.com/hashicorp/go-multierror"
	amqp "github.com/rabbitmq/amqp091-go"
	"net"
	"sync"
)

type Channel interface {
	Publish(ctx context.Context, exchange, routingKey string, payload *Payload, headers map[string]any) (uint64, error)
	Close() error
}

type ChannelOptions struct {
	publisherConfirms bool
}

type ChannelOptionFunc func(*ChannelOptions)

func WithChannelPublisherConfirms() func(*ChannelOptions) {
	return func(co *ChannelOptions) {
		co.publisherConfirms = true
	}
}

func NewChannel(url string, options ...ChannelOptionFunc) (Channel, error) {
	opts := &ChannelOptions{}
	for _, o := range options {
		o(opts)
	}
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("cannot connect producer to '%s': %w", url, err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("cannot create producer channel %w", err)
	}
	result := &channel{
		conn:              conn,
		ch:                ch,
		publisherConfirms: opts.publisherConfirms,
	}
	if opts.publisherConfirms {
		err = result.confirm()
		if err != nil {
			var rerror error
			rerror = multierror.Append(rerror, err)
			if err = result.Close(); err != nil {
				rerror = multierror.Append(rerror, err)
			}
			return nil, rerror
		}
	}
	return result, nil
}

type channel struct {
	closeOnce sync.Once

	conn *amqp.Connection
	ch   *amqp.Channel

	publisherConfirms bool
}

func (c *channel) Close() (err error) {
	c.closeOnce.Do(
		func() {
			var closer util.MultiCloser
			closer.Add(c.ch.Close)
			closer.Add(c.conn.Close)
			err = closer.Close()
		},
	)
	return
}

func (c *channel) confirm() error {
	err := c.ch.Confirm(false)
	if err != nil {
		return fmt.Errorf("cannot put channel into confirm mode %w", err)
	}
	return nil
}

func (c *channel) Publish(ctx context.Context, exchange, routingKey string, payload *Payload, headers map[string]any) (uint64, error) {
	msg := toMessage(payload, headers)
	if c.publisherConfirms {
		return c.publishAndConfirm(ctx, exchange, routingKey, &msg)
	} else {
		return 0, c.publish(ctx, exchange, routingKey, &msg)
	}
}

func (c *channel) publish(ctx context.Context, exchange, routingKey string, msg *amqp.Publishing) error {
	err := c.ch.PublishWithContext(
		ctx,
		exchange,   // publish to an exchange
		routingKey, // routing to 0 or more queues
		true,       // mandatory
		false,      // immediate
		*msg,
	)
	if err != nil {
		return fmt.Errorf("publish failed: %w", err)
	}
	return nil
}

func (c *channel) publishWithDeferredConfirm(ctx context.Context, exchange string, routingKey string, msg *amqp.Publishing) (*amqp.DeferredConfirmation, error) {
	d, err := c.ch.PublishWithDeferredConfirmWithContext(
		ctx,
		exchange,   // publish to an exchange
		routingKey, // routing to 0 or more queues
		true,       // mandatory
		false,      // immediate
		*msg,
	)
	if err != nil {
		return nil, fmt.Errorf("publish confirm failed: %w", err)
	}
	return d, nil
}

func (c *channel) publishAndConfirm(ctx context.Context, exchange, routingKey string, msg *amqp.Publishing) (uint64, error) {
	d, err := c.publishWithDeferredConfirm(ctx, exchange, routingKey, msg)
	if err != nil {
		return 0, err
	}
	ack, err := d.WaitContext(ctx)
	if err != nil {
		return 0, fmt.Errorf("publisher confirmation failed: %w", err)
	}
	if !ack {
		return d.DeliveryTag, fmt.Errorf("message nacked  %v", d.DeliveryTag)
	}
	return d.DeliveryTag, nil
}

func toTable(vs map[string]interface{}) amqp.Table {
	result := amqp.Table{}
	if len(vs) != 0 {
		for k, v := range vs {
			result[k] = v
		}
	}
	return result
}

func toMessage(payload *Payload, headers map[string]any) amqp.Publishing {
	return amqp.Publishing{
		Headers:         toTable(headers),
		ContentType:     payload.ContentType,
		ContentEncoding: payload.ContentEncoding,
		Body:            payload.Body,
		DeliveryMode:    amqp.Persistent, // 1=non-persistent, 2=persistent
	}
}

func shouldRetry(err error) bool {
	cause := errors.Unwrap(err)
	if cause != nil {
		err = cause
	}
	switch t := err.(type) {
	case *amqp.Error:
		return t.Code == amqp.ChannelError || t.Code == amqp.FrameError
	case amqp.Error:
		return t.Code == amqp.ChannelError || t.Code == amqp.FrameError
	case net.Error, *net.OpError:
		return true
	}
	return false
}
