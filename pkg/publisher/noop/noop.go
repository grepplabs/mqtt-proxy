package noop

import (
	"context"
	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/grepplabs/mqtt-proxy/pkg/runtime"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/atomic"
)

const (
	publisherName = "noop"
)

type Publisher struct {
	done    *runtime.DoneChannel
	counter atomic.Int32
	logger  log.Logger
}

func New(logger log.Logger, _ *prometheus.Registry) *Publisher {
	return &Publisher{
		done:   runtime.NewDoneChannel(),
		logger: logger.WithField("publisher", publisherName),
	}
}

func (s Publisher) Name() string {
	return publisherName
}

func (p *Publisher) Publish(_ context.Context, request *apis.PublishRequest) (*apis.PublishResponse, error) {
	if request == nil {
		return nil, errors.New("Empty request")
	}

	p.logger.Debugf("sync publish: messageID=%v", request.MessageID)

	return &apis.PublishResponse{
		ID: p.counter.Inc(),
	}, nil
}

func (p *Publisher) PublishAsync(_ context.Context, request *apis.PublishRequest, callback apis.PublishCallbackFunc) error {
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

func (p *Publisher) Serve() error {
	defer p.logger.Infof("Serve stopped")

	<-p.done.Done()
	return nil
}

func (p *Publisher) Shutdown(err error) {
	defer p.logger.WithError(err).Infof("internal server shutdown")
	defer p.Close()

	p.done.Close()
	return
}

func (p *Publisher) Close() error {
	defer p.logger.Infof("publisher closed")

	p.done.Close()
	return nil
}
