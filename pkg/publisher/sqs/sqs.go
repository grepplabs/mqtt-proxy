package sqs

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/grepplabs/mqtt-proxy/pkg/util"
	"strconv"
	"strings"
	"sync"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/grepplabs/mqtt-proxy/pkg/runtime"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	mqttQosAttribute    = "mqtt.qos"
	mqttDupAttribute    = "mqtt.dup"
	mqttRetainAttribute = "mqtt.retain"
	mqttMsgIDAttribute  = "mqtt.packet.id"
	mqttMsgFmtAttribute = "mqtt.fmt"
)

const (
	publisherName = "sqs"
)

type Publisher struct {
	done   *runtime.DoneChannel
	logger log.Logger
	client *sqs.Client
	opts   options

	queueUrls sync.Map
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
	client, err := newClient(logger, options)
	if err != nil {
		return nil, err
	}
	publisher := &Publisher{
		logger: logger,
		done:   runtime.NewDoneChannel(),
		client: client,
		opts:   options,
	}
	return publisher, nil
}

func (p *Publisher) Name() string {
	return publisherName
}

func (p *Publisher) getGetQueueURL(ctx context.Context, mqttTopic string) (*string, error) {
	queueName, err := p.findQueueName(mqttTopic)
	if err != nil {
		return nil, err
	}
	value, ok := p.queueUrls.Load(queueName)
	if ok && value != nil {
		return value.(*string), nil
	}
	output, err := p.client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})
	if err != nil {
		return nil, fmt.Errorf("GetQueueUrl '%s' failed: %w", queueName, err)
	}
	queueUrl := output.QueueUrl
	p.queueUrls.Store(queueName, queueUrl)
	return queueUrl, nil
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
	return "", fmt.Errorf("sqs queue not found for MQTT topic %s", mqttTopic)
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

func (p *Publisher) sendMessage(ctx context.Context, request *apis.PublishRequest) (*apis.PublishResponse, error) {
	if request == nil {
		return nil, errors.New("empty request")
	}
	queueURL, err := p.getGetQueueURL(ctx, request.TopicName)
	if err != nil {
		return nil, err
	}
	messageBody, err := util.GetMessageBody(p.opts.messageFormat, request)
	if err != nil {
		return nil, err
	}
	messageID := aws.String(strconv.FormatUint(uint64(request.MessageID), 10))
	input := &sqs.SendMessageInput{
		MessageAttributes: map[string]types.MessageAttributeValue{
			mqttDupAttribute: {
				DataType:    aws.String("String.bool"),
				StringValue: aws.String(strconv.FormatBool(request.Dup)),
			},
			mqttQosAttribute: {
				DataType:    aws.String("Number"),
				StringValue: aws.String(strconv.FormatUint(uint64(request.Qos), 10)),
			},
			mqttRetainAttribute: {
				DataType:    aws.String("String.bool"),
				StringValue: aws.String(strconv.FormatBool(request.Retain)),
			},
			mqttMsgIDAttribute: {
				DataType:    aws.String("Number"),
				StringValue: messageID,
			},
			mqttMsgFmtAttribute: {
				DataType:    aws.String("String"),
				StringValue: aws.String(p.opts.messageFormat),
			},
		},
		MessageBody: aws.String(string(messageBody)),
		QueueUrl:    queueURL,
	}
	if strings.HasSuffix(*queueURL, ".fifo") {
		input.MessageGroupId = aws.String(p.GetMessageGroupId(request))
		input.MessageDeduplicationId = messageID
	}
	output, err := p.client.SendMessage(ctx, input)
	var publishID apis.PublishID
	if err == nil {
		publishID = aws.ToString(output.MessageId)
	}
	return &apis.PublishResponse{
		ID:    publishID,
		Error: err,
	}, nil
}

func (p *Publisher) GetMessageGroupId(request *apis.PublishRequest) string {
	messageGroupId := request.ClientID
	if messageGroupId == "" {
		messageGroupId = "mqtt-proxy"
	}
	return messageGroupId
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
	defer p.logger.Infof("sqs publisher closed")

	p.done.Close()

	return nil
}

func newClient(logger log.Logger, options options) (*sqs.Client, error) {
	opts := make([]func(*awsconfig.LoadOptions) error, 0)
	opts = append(opts, awsconfig.WithRegion(options.region))
	opts = append(opts, awsconfig.WithSharedConfigProfile(options.profile))
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	stsClient := sts.NewFromConfig(cfg)
	output, err := stsClient.GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get caller identity for SQS client %w", err)
	}
	logger.Infof("Creating SQS client with identity %s", aws.ToString(output.UserId))
	return sqs.NewFromConfig(cfg), nil
}
