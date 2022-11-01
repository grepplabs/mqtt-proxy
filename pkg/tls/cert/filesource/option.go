package filesource

import (
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"time"
)

type Option func(*fileSource)

func WithLogger(logger log.Logger) Option {
	return func(c *fileSource) {
		c.logger = logger
	}
}

func WithX509KeyPair(certFile, keyFile string) Option {
	return func(c *fileSource) {
		c.certFile = certFile
		c.keyFile = keyFile
	}
}

func WithClientAuthFile(clientAuthFile string) Option {
	return func(c *fileSource) {
		c.clientAuthFile = clientAuthFile
	}
}

func WithClientCRLFile(clientCRLFile string) Option {
	return func(c *fileSource) {
		c.clientCRLFile = clientCRLFile
	}
}

func WithRefresh(refresh time.Duration) Option {
	return func(c *fileSource) {
		c.refresh = refresh
	}
}

func WithNotifyFunc(notifyFunc func()) Option {
	return func(c *fileSource) {
		c.notifyFunc = notifyFunc
	}
}
