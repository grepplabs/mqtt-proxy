package mqtt

import (
	"crypto/tls"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	mqtthandler "github.com/grepplabs/mqtt-proxy/pkg/mqtt/handler"
	"github.com/grepplabs/mqtt-proxy/pkg/prober"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	a := assert.New(t)

	logger := log.NewDefaultLogger()
	registry := prometheus.NewRegistry()
	serverProber := prober.NewHTTP()
	tlsCfg := &tls.Config{}
	handler := &mqtthandler.MQTTHandler{}

	server := New(logger, registry, serverProber,
		WithNetwork("tcp"),
		WithListen("0.0.0.0:1883"),
		WithReadTimeout(10*time.Second),
		WithWriteTimeout(11*time.Second),
		WithIdleTimeout(12*time.Second),
		WithReaderBufferSize(2048),
		WithWriterBufferSize(4096),
		WithTLSConfig(tlsCfg),
		WithHandler(handler),
	)

	a.NotNil(server.logger)
	a.NotNil(server.opts)
	a.NotNil(server.srv)
	a.NotNil(server.mux)
	a.Same(serverProber, server.prober)

	a.Equal("tcp", server.opts.network)
	a.Equal("0.0.0.0:1883", server.opts.listen)
	a.Equal(10*time.Second, server.opts.readTimeout)
	a.Equal(11*time.Second, server.opts.writeTimeout)
	a.Equal(12*time.Second, server.opts.idleTimeout)
	a.Equal(2048, server.opts.readerBufferSize)
	a.Equal(4096, server.opts.writerBufferSize)
	a.Equal(handler, server.opts.handler)

	a.Equal("tcp", server.srv.Network)
	a.Equal("0.0.0.0:1883", server.srv.Addr)
	a.Equal(10*time.Second, server.srv.ReadTimeout)
	a.Equal(11*time.Second, server.srv.WriteTimeout)
	a.Equal(12*time.Second, server.srv.IdleTimeout)
	a.Equal(2048, server.srv.ReaderBufferSize)
	a.Equal(4096, server.srv.WriterBufferSize)
	a.NotNil(server.srv.ErrorLog)
	a.Equal(handler, server.srv.Handler)

}
