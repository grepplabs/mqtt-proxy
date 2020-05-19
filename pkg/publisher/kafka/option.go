package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/pkg/errors"
	"time"
)

type options struct {
	bootstrapServers string
	gracePeriod      time.Duration
	workers          int
	// see https://github.com/edenhill/librdkafka/blob/master/CONFIGURATION.md
	configMap kafka.ConfigMap
}

func (o options) validate() error {
	if o.bootstrapServers == "" {
		return errors.New("kafka.bootstrap-servers must not be empty")
	}
	if o.workers < 1 {
		return errors.New("kafka.workers must be greater than 0")
	}
	return nil
}

type Option interface {
	apply(*options)
}

type optionFunc func(*options)

func (f optionFunc) apply(o *options) {
	f(o)
}

func WithGracePeriod(t time.Duration) Option {
	return optionFunc(func(o *options) {
		o.gracePeriod = t
	})
}

func WithBootstrapServers(s string) Option {
	return optionFunc(func(o *options) {
		o.bootstrapServers = s
	})
}

func WithConfigMap(configMap kafka.ConfigMap) Option {
	return optionFunc(func(o *options) {
		o.configMap = configMap
	})
}

func WithWorkers(v int) Option {
	return optionFunc(func(o *options) {
		o.workers = v
	})
}
