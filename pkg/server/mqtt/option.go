package mqtt

import (
	"crypto/tls"
	"time"

	mqttserver "github.com/grepplabs/mqtt-proxy/pkg/mqtt/server"
)

type options struct {
	gracePeriod  time.Duration
	listen       string
	network      string
	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration

	readerBufferSize int
	writerBufferSize int

	tlsConfig *tls.Config

	handler mqttserver.Handler
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

func WithListen(s string) Option {
	return optionFunc(func(o *options) {
		o.listen = s
	})
}

func WithNetwork(s string) Option {
	return optionFunc(func(o *options) {
		o.network = s
	})
}

func WithTLSConfig(cfg *tls.Config) Option {
	return optionFunc(func(o *options) {
		o.tlsConfig = cfg
	})
}

func WithHandler(handler mqttserver.Handler) Option {
	return optionFunc(func(o *options) {
		o.handler = handler
	})
}

func WithReadTimeout(d time.Duration) Option {
	return optionFunc(func(o *options) {
		o.readTimeout = d
	})
}

func WithWriteTimeout(d time.Duration) Option {
	return optionFunc(func(o *options) {
		o.writeTimeout = d
	})
}

func WithIdleTimeout(d time.Duration) Option {
	return optionFunc(func(o *options) {
		o.idleTimeout = d
	})
}

func WithReaderBufferSize(i int) Option {
	return optionFunc(func(o *options) {
		o.readerBufferSize = i
	})
}

func WithWriterBufferSize(i int) Option {
	return optionFunc(func(o *options) {
		o.writerBufferSize = i
	})
}
