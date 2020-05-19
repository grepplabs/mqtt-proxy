package prober

import (
	"io"
	"net/http"

	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"go.uber.org/atomic"
)

type check func() bool

type HTTPProbe struct {
	ready   atomic.Bool
	healthy atomic.Bool
}

func NewHTTP() *HTTPProbe {
	return &HTTPProbe{}
}

func (p *HTTPProbe) HealthyHandler(logger log.Logger) http.HandlerFunc {
	return p.handler(logger, p.isHealthy)
}

func (p *HTTPProbe) ReadyHandler(logger log.Logger) http.HandlerFunc {
	return p.handler(logger, p.isReady)
}

func (p *HTTPProbe) handler(logger log.Logger, c check) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if !c() {
			http.Error(w, "NOT OK", http.StatusServiceUnavailable)
			return
		}
		if _, err := io.WriteString(w, "OK"); err != nil {
			logger.WithError(err).Errorf("failed to write probe response")
		}
	}
}

func (p *HTTPProbe) isReady() bool {
	return p.ready.Load()
}

func (p *HTTPProbe) isHealthy() bool {
	return p.healthy.Load()
}

func (p *HTTPProbe) Ready() {
	p.ready.Store(true)
}

func (p *HTTPProbe) NotReady(_ error) {
	p.ready.Store(false)
}

func (p *HTTPProbe) Healthy() {
	p.healthy.Store(true)
}

func (p *HTTPProbe) NotHealthy(_ error) {
	p.healthy.Store(false)
}
