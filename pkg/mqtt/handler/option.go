package mqtthandler

type options struct {
	allowUnauthenticated bool
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
