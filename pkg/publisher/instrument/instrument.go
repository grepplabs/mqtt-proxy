package instrument

import (
	"context"
	"strconv"
	"time"

	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	syncType  = "sync"
	asyncType = "async"
)

type Publisher struct {
	delegate apis.Publisher
	metrics  *publisherMetrics
}

type publisherMetrics struct {
	publishDuration *prometheus.HistogramVec
}

func New(delegate apis.Publisher, registry *prometheus.Registry) *Publisher {
	return &Publisher{
		delegate: delegate,
		metrics:  newPublisherMetrics(delegate.Name(), registry),
	}
}

func (p *Publisher) Name() string {
	return p.delegate.Name()
}

func (p *Publisher) Publish(context context.Context, publishRequest *apis.PublishRequest) (*apis.PublishResponse, error) {
	start := time.Now()
	publishResponse, err := p.delegate.Publish(context, publishRequest)
	p.metrics.publishDuration.WithLabelValues(syncType, strconv.Itoa(int(publishRequest.Qos))).Observe(time.Since(start).Seconds())
	return publishResponse, err
}

func (p *Publisher) PublishAsync(context context.Context, publishRequest *apis.PublishRequest, callback apis.PublishCallbackFunc) error {
	start := time.Now()
	return p.delegate.PublishAsync(context, publishRequest, func(request *apis.PublishRequest, response *apis.PublishResponse) {
		p.metrics.publishDuration.WithLabelValues(asyncType, strconv.Itoa(int(publishRequest.Qos))).Observe(time.Since(start).Seconds())
		callback(request, response)
	})
}

func (p *Publisher) Serve() error {
	return p.delegate.Serve()
}

func (p *Publisher) Close() error {
	return p.delegate.Close()
}

func (p *Publisher) Shutdown(err error) {
	p.delegate.Shutdown(err)
}

func newPublisherMetrics(name string, registry *prometheus.Registry) *publisherMetrics {
	publishDuration := promauto.With(registry).NewHistogramVec(prometheus.HistogramOpts{
		Name:        "mqtt_proxy_publisher_publish_duration_seconds",
		Help:        "Tracks the latencies for publish requests.",
		ConstLabels: prometheus.Labels{"name": name},
	}, []string{"type", "qos"})

	return &publisherMetrics{
		publishDuration: publishDuration,
	}
}
