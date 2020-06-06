package http

import (
	"github.com/grepplabs/mqtt-proxy/pkg/log"
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

	server := New(logger, registry, serverProber,
		WithListen("0.0.0.0:1883"),
		WithGracePeriod(10*time.Second),
	)

	a.NotNil(server.logger)
	a.NotNil(server.opts)
	a.NotNil(server.srv)
	a.NotNil(server.mux)
	a.Same(serverProber, server.prober)

	a.Equal("0.0.0.0:1883", server.opts.listen)
	a.Equal(10*time.Second, server.opts.gracePeriod)

	a.Same(server.mux, server.srv.Handler)
	a.Equal(server.opts.listen, server.srv.Addr)
}
