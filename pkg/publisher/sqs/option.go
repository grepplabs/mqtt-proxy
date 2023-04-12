package sqs

import (
	"errors"
	"github.com/grepplabs/mqtt-proxy/pkg/config"
)

type options struct {
	region  string
	profile string

	defaultQueue  string
	queueMappings config.TopicMappings
	messageFormat string
}

type Option interface {
	apply(*options)
}

func (o options) validate() error {
	if o.defaultQueue == "" && len(o.queueMappings.Mappings) == 0 {
		return errors.New("sqs default queue or queue mappings must be provided")
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

func WithQueueMappings(queueMappings config.TopicMappings) Option {
	return optionFunc(func(o *options) {
		o.queueMappings = queueMappings
	})
}

func WithDefaultQueue(s string) Option {
	return optionFunc(func(o *options) {
		o.defaultQueue = s
	})
}

func WithMessageFormat(s string) Option {
	return optionFunc(func(o *options) {
		o.messageFormat = s
	})
}
