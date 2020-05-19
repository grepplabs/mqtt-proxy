package noop

import (
	"context"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/pkg/errors"

	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/grepplabs/mqtt-proxy/pkg/runtime"
	"go.uber.org/atomic"
)

type NoopPublisher struct {
	done    *runtime.DoneChannel
	counter atomic.Int32
	logger  log.Logger
}

func New(logger log.Logger) (*NoopPublisher, error) {
	return &NoopPublisher{
		done:   runtime.NewDoneChannel(),
		logger: logger.WithField("publisher", "noop"),
	}, nil
}

func (p *NoopPublisher) Publish(_ context.Context, request *apis.PublishRequest) (*apis.PublishResponse, error) {
	if request == nil {
		return nil, errors.New("Empty request")
	}

	p.logger.Debugf("sync publish: messageID=%v", request.MessageID)

	return &apis.PublishResponse{
		ID: p.counter.Inc(),
	}, nil
}

func (p *NoopPublisher) PublishAsync(_ context.Context, request *apis.PublishRequest, callback apis.PublishCallbackFunc) error {
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
