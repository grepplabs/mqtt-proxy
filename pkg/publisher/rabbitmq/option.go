package rabbitmq

import (
	"errors"
	"github.com/grepplabs/mqtt-proxy/pkg/config"
	"time"
)

type options struct {
	scheme            string
	host              string
	port              int
	username          string
	password          string
	vhost             string
	connectionTimeout time.Duration
	requestTimeout    time.Duration

	exchange                     string
	defaultQueue                 string
	queueMappings                config.TopicMappings
	messageFormat                string
	publisherConfirmsAtMostOnce  bool
	publisherConfirmsAtLeastOnce bool
	publisherConfirmsExactlyOnce bool
}

type Option interface {
	apply(*options)
}

func (o options) validate() error {
	if o.scheme == "" {
		return errors.New("parameter rabbitmq.scheme is required")
	}
	if o.host == "" {
		return errors.New("parameter rabbitmq.host is required")
	}
	if o.port <= 0 {
		return errors.New("parameter rabbitmq.port is required")
	}
	if o.vhost == "" {
		return errors.New("parameter rabbitmq.vhost is required")
	}
	if (o.username != "" && o.password == "") || (o.username == "" && o.password != "") {
		return errors.New("both parameter rabbitmq.username and rabbitmq.password are required")
	}
	if o.defaultQueue == "" && len(o.queueMappings.Mappings) == 0 {
		return errors.New("rabbitmq default queue or queue mappings must be provided")
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

func WithScheme(scheme string) Option {
	return optionFunc(func(o *options) {
		o.scheme = scheme
	})
}

func WithHost(host string) Option {
	return optionFunc(func(o *options) {
		o.host = host
	})
}

func WithPort(port int) Option {
	return optionFunc(func(o *options) {
		o.port = port
	})
}

func WithUsername(username string) Option {
	return optionFunc(func(o *options) {
		o.username = username
	})
}

func WithPassword(password string) Option {
	return optionFunc(func(o *options) {
		o.password = password
	})
}

func WithVHost(vhost string) Option {
	return optionFunc(func(o *options) {
		o.vhost = vhost
	})
}

func WithConnectionTimeout(connectionTimeout time.Duration) Option {
	return optionFunc(func(o *options) {
		o.connectionTimeout = connectionTimeout
	})
}

func WithRequestTimeout(requestTimeout time.Duration) Option {
	return optionFunc(func(o *options) {
		o.requestTimeout = requestTimeout
	})
}

func WithExchange(exchange string) Option {
	return optionFunc(func(o *options) {
		o.exchange = exchange
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

func WithPublisherConfirmsAtMostOnce(b bool) Option {
	return optionFunc(func(o *options) {
		o.publisherConfirmsAtMostOnce = b
	})
}

func WithPublisherConfirmsAtLeastOnce(b bool) Option {
	return optionFunc(func(o *options) {
		o.publisherConfirmsAtLeastOnce = b
	})
}

func WithPublisherConfirmsExactlyOnce(b bool) Option {
	return optionFunc(func(o *options) {
		o.publisherConfirmsExactlyOnce = b
	})
}
