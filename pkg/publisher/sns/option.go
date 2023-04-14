package sqs

import (
	"errors"
	"github.com/grepplabs/mqtt-proxy/pkg/config"
)

type options struct {
	region  string
	profile string

	defaultTopicARN  string
	topicARNMappings config.TopicMappings
	messageFormat    string
}

type Option interface {
	apply(*options)
}

func (o options) validate() error {
	if o.defaultTopicARN == "" && len(o.topicARNMappings.Mappings) == 0 {
		return errors.New("sns default topic arn or topic arn mappings must be provided")
	}
	if o.messageFormat == "" {
		return errors.New("publisher message format must not be empty")
	}
	return nil
}

type optionFunc func(*options)

func (f optionFunc) apply(o *options) {
	f(o)
}

func WithAWSRegion(s string) Option {
	return optionFunc(func(o *options) {
		o.region = s
	})
}

func WithAWSProfile(s string) Option {
	return optionFunc(func(o *options) {
		o.profile = s
	})
}

func WithTopicARNMappings(queueMappings config.TopicMappings) Option {
	return optionFunc(func(o *options) {
		o.topicARNMappings = queueMappings
	})
}

func WithDefaultTopicARN(s string) Option {
	return optionFunc(func(o *options) {
		o.defaultTopicARN = s
	})
}

func WithMessageFormat(s string) Option {
	return optionFunc(func(o *options) {
		o.messageFormat = s
	})
}
