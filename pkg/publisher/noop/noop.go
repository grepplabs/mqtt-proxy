package noop

import (
	"context"

	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/grepplabs/mqtt-proxy/pkg/runtime"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/atomic"
)

const (
	syncType  string = "sync"
	asyncType string = "async"
)

type NoopPublisher struct {
	done    *runtime.DoneChannel
	counter atomic.Int32
	logger  log.Logger
	metrics *noopMetrics
}

type noopMetrics struct {
	requestsTotal  *prometheus.CounterVec
	responsesTotal *prometheus.CounterVec
}

func New(logger log.Logger, registry *prometheus.Registry) (*NoopPublisher, error) {
	return &NoopPublisher{
		done:    runtime.NewDoneChannel(),
		logger:  logger.WithField("publisher", "noop"),
		metrics: newNoopMetrics(registry),
	}, nil
}

func (p *NoopPublisher) Publish(_ context.Context, request *apis.PublishRequest) (*apis.PublishResponse, error) {
	p.metrics.requestsTotal.WithLabelValues(syncType).Inc()

	if request == nil {
		return nil, errors.New("Empty request")
	}

	p.logger.Debugf("sync publish: messageID=%v", request.MessageID)

	return &apis.PublishResponse{
		ID: p.counter.Inc(),
	}, nil
}

func (p *NoopPublisher) PublishAsync(_ context.Context, request *apis.PublishRequest, callback apis.PublishCallbackFunc) error {
	p.metrics.requestsTotal.WithLabelValues(asyncType).Inc()

	if request == nil || callback == nil {
		return errors.New("Empty request/callback")
	}

	p.logger.Debugf("async publish: messageID=%v", request.MessageID)

	go func() {
		id := p.counter.Inc()
		p.logger.Debugf("async response: messageID=%v ,id=%v", request.MessageID, id)

		callback(request, &apis.PublishResponse{
			ID: id,
		})
	}()
	return nil
}

func (p *NoopPublisher) Serve() error {
	defer p.logger.Infof("Serve stopped")

	<-p.done.Done()
	return nil
}

func (p *NoopPublisher) Shutdown(err error) {
	defer p.logger.WithError(err).Infof("internal server shutdown")
	defer p.Close()

	p.done.Close()
	return
}

func (p *NoopPublisher) Close() error {
	defer p.logger.Infof("publisher closed")

	p.done.Close()
	return nil
}

func newNoopMetrics(registry *prometheus.Registry) *noopMetrics {
	requestsTotal := promauto.With(registry).NewCounterVec(prometheus.CounterOpts{
		Name:        "mqtt_proxy_publisher_requests_total",
		Help:        "Total number of publish requests.",
		ConstLabels: prometheus.Labels{"name": "noop"},
	}, []string{"type"})

	requestsTotal.WithLabelValues(syncType)
	requestsTotal.WithLabelValues(asyncType)

	responsesTotal := promauto.With(registry).NewCounterVec(prometheus.CounterOpts{
		Name:        "mqtt_proxy_publisher_responses_total",
		Help:        "Total number of publish responses.",
		ConstLabels: prometheus.Labels{"name": "noop"},
	}, []string{"type"})

	responsesTotal.WithLabelValues(syncType)
	responsesTotal.WithLabelValues(asyncType)

	return &noopMetrics{
		requestsTotal:  requestsTotal,
		responsesTotal: responsesTotal,
	}
}
