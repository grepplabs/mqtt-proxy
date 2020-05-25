package mqtthandler

import "time"

type options struct {
	ignoreUnsupported       []string
	allowUnauthenticated    []string
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

func WithIgnoreUnsupported(a []string) Option {
	return optionFunc(func(o *options) {
		o.ignoreUnsupported = a
	})
}

func WithAllowUnauthenticated(a []string) Option {
	return optionFunc(func(o *options) {
		o.allowUnauthenticated = a
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
