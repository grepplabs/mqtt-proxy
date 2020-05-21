package mqtthandler

import "time"

type options struct {
	allowUnauthenticated    bool
	publishTimeout          time.Duration
	publishAsyncAtMostOnce  bool
	publishAsyncAtLeastOnce bool
	publishAsyncExactlyOnce bool
}

type Option interface {
	apply(*options)
}

type optionFunc func(*options)

func (f optionFunc) apply(o *options) {
	f(o)
}

func WithAllowUnauthenticated(b bool) Option {
	return optionFunc(func(o *options) {
		o.allowUnauthenticated = b
	})
}

func WithPublishTimeout(d time.Duration) Option {
	return optionFunc(func(o *options) {
		o.publishTimeout = d
	})
}

func WithPublishAsyncAtMostOnce(b bool) Option {
	return optionFunc(func(o *options) {
		o.publishAsyncAtMostOnce = b
	})
}

func WithPublishAsyncAtLeastOnce(b bool) Option {
	return optionFunc(func(o *options) {
		o.publishAsyncAtLeastOnce = b
	})
}

func WithPublishAsyncExactlyOnce(b bool) Option {
	return optionFunc(func(o *options) {
		o.publishAsyncExactlyOnce = b
	})
}
