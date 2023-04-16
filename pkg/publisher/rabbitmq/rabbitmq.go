package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/grepplabs/mqtt-proxy/pkg/config"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
	"github.com/grepplabs/mqtt-proxy/pkg/runtime"
	"github.com/grepplabs/mqtt-proxy/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
)

const (
	mqttQosHeader    = "mqtt.qos"
	mqttDupHeader    = "mqtt.dup"
	mqttRetainHeader = "mqtt.retain"
	mqttMsgIDHeader  = "mqtt.packet.id"
	mqttMsgFmtHeader = "mqtt.fmt"
)
const (
	publisherName = "rabbitmq"
)

type Publisher struct {
	done    *runtime.DoneChannel
	logger  log.Logger
	clients map[byte]Client
	opts    options
}

func New(logger log.Logger, _ *prometheus.Registry, opts ...Option) (*Publisher, error) {
	logger = logger.WithField("publisher", publisherName)

	options := options{}
	for _, o := range opts {
		o.apply(&options)
	}
	err := options.validate()
	if err != nil {
		return nil, err
	}
	clients, err := newClients(logger, options)
	if err != nil {
		return nil, err
	}
	publisher := &Publisher{
		logger:  logger,
		done:    runtime.NewDoneChannel(),
		clients: clients,
		opts:    options,
	}
	return publisher, nil
}

func (p *Publisher) Name() string {
	return publisherName
}

func (p *Publisher) Publish(ctx context.Context, request *apis.PublishRequest) (*apis.PublishResponse, error) {
	response, err := p.sendMessage(ctx, request)
	if err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, response.Error
	}
	return response, nil
}

func (p *Publisher) PublishAsync(ctx context.Context, request *apis.PublishRequest, callback apis.PublishCallbackFunc) error {
	if request == nil || callback == nil {
		return errors.New("empty request/callback")
	}
	response, err := p.sendMessage(ctx, request)
	if err != nil {
		return err
	}
	callback(request, response)
	return nil
}

func (p *Publisher) findQueueName(mqttTopic string) (string, error) {
	for _, mapping := range p.opts.queueMappings.Mappings {
		if mapping.RegExp.MatchString(mqttTopic) {
			return mapping.Topic, nil
		}
	}
	if p.opts.defaultQueue != "" {
		return p.opts.defaultQueue, nil
	}
	return "", fmt.Errorf("rabbitmq queue not found for MQTT topic %s", mqttTopic)
}

func (p *Publisher) sendMessage(ctx context.Context, request *apis.PublishRequest) (*apis.PublishResponse, error) {
	client := p.clients[request.Qos]
	if client == nil {
		return nil, fmt.Errorf("rabbitmq client for qos %d not found", request.Qos)
	}
	routingKey, err := p.findQueueName(request.TopicName)
	if err != nil {
		return nil, err
	}
	payload, err := p.getPayload(request)
	if err != nil {
		return nil, err
	}
	headers := p.getHeaders(request)
	deliveryTag, err := client.Publish(ctx, p.opts.exchange, routingKey, payload, headers)
	if err != nil {
		return nil, err
	}
	return &apis.PublishResponse{
		ID:    deliveryTag,
		Error: nil,
	}, nil
}

func (p *Publisher) getHeaders(request *apis.PublishRequest) map[string]any {
	headers := map[string]any{
		mqttQosHeader:    strconv.FormatUint(uint64(request.Qos), 10),
		mqttDupHeader:    strconv.FormatBool(request.Dup),
		mqttRetainHeader: strconv.FormatBool(request.Retain),
		mqttMsgIDHeader:  strconv.FormatUint(uint64(request.MessageID), 10),
		mqttMsgFmtHeader: p.opts.messageFormat,
	}
	return headers
}

func (p *Publisher) getPayload(request *apis.PublishRequest) (*Payload, error) {
	messageBody, err := util.GetMessageBody(p.opts.messageFormat, request)
	if err != nil {
		return nil, err
	}
	payload := &Payload{Body: messageBody}
	switch p.opts.messageFormat {
	case config.MessageFormatPlain:
		payload.ContentType = "text/plain"
	case config.MessageFormatBase64:
		payload.ContentType = "text/plain"
		payload.ContentEncoding = "base64"
	case config.MessageFormatJson:
		payload.ContentType = "application/json"
	}
	return payload, nil
}

func (p *Publisher) Serve() error {
	defer p.logger.Infof("Serve stopped")

	<-p.done.Done()
	return nil
}

func (p *Publisher) Shutdown(err error) {
	defer p.logger.WithError(err).Infof("internal server shutdown")
	defer p.Close()

	p.done.Close()
	return
}

func (p *Publisher) Close() error {
	defer p.logger.Infof("rabbitmq publisher closed")

	p.done.Close()

	var rerror error
	for _, p := range p.clients {
		if err := p.Close(); err != nil {
			rerror = multierror.Append(rerror, err)
		}
	}
	return rerror
}

func newClients(logger log.Logger, opts options) (map[byte]Client, error) {
	clients := make(map[byte]Client)

	var closer util.MultiCloser

	atMostOnceClient, err := NewClient(logger, opts.publisherConfirmsAtMostOnce, opts)
	closer.Add(atMostOnceClient.Close)
	if err != nil {
		_ = closer.Close()
		return nil, err
	}
	clients[mqttproto.AT_MOST_ONCE] = atMostOnceClient

	atLeastOnceClient, err := NewClient(logger, opts.publisherConfirmsAtLeastOnce, opts)
	closer.Add(atLeastOnceClient.Close)
	if err != nil {
		_ = closer.Close()
		return nil, err
	}
	clients[mqttproto.AT_LEAST_ONCE] = atLeastOnceClient

	exactlyOnceClient, err := NewClient(logger, opts.publisherConfirmsExactlyOnce, opts)
	closer.Add(exactlyOnceClient.Close)
	if err != nil {
		_ = closer.Close()
		return nil, err
	}
	clients[mqttproto.EXACTLY_ONCE] = exactlyOnceClient
	return clients, nil
}
