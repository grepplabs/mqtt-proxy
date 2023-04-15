package sqs

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/grepplabs/mqtt-proxy/pkg/util"
	"strconv"
	"strings"

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
	publisherName = "sns"
)

type Publisher struct {
	done   *runtime.DoneChannel
	logger log.Logger
	client *sns.Client
	opts   options
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

func (p *Publisher) findTopicARN(mqttTopic string) (string, error) {
	for _, mapping := range p.opts.topicARNMappings.Mappings {
		if mapping.RegExp.MatchString(mqttTopic) {
			return mapping.Topic, nil
		}
	}
	if p.opts.defaultTopicARN != "" {
		return p.opts.defaultTopicARN, nil
	}
	return "", fmt.Errorf("sns topic ARN not found for MQTT topic %s", mqttTopic)
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
	topicARN, err := p.findTopicARN(request.TopicName)
	if err != nil {
		return nil, err
	}
	messageBody, err := util.GetMessageBody(p.opts.messageFormat, request)
	if err != nil {
		return nil, err
	}
	messageID := aws.String(strconv.FormatUint(uint64(request.MessageID), 10))

	input := &sns.PublishInput{
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
		Message:  aws.String(string(messageBody)),
		TopicArn: aws.String(topicARN),
	}
	if strings.HasSuffix(topicARN, ".fifo") {
		input.MessageGroupId = aws.String(p.GetMessageGroupId(request))
		input.MessageDeduplicationId = messageID
	}
	output, err := p.client.Publish(ctx, input)
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
	defer p.logger.Infof("sns publisher closed")

	p.done.Close()

	return nil
}

func newClient(logger log.Logger, options options) (*sns.Client, error) {
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
		return nil, fmt.Errorf("failed to get caller identity for SNS client %w", err)
	}
	logger.Infof("Creating SNS client with identity %s", aws.ToString(output.UserId))
	return sns.NewFromConfig(cfg), nil
}
