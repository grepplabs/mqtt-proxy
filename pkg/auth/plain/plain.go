package plain

import (
	"context"
	"fmt"

	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	authName = "plain"
)

type plainAuthenticator struct {
	logger log.Logger
	opts   options
}

func New(logger log.Logger, _ *prometheus.Registry, opts ...Option) (apis.UserPasswordAuthenticator, error) {
	options := options{
		credentials: make(map[string]string),
	}

	for _, o := range opts {
		err := o.apply(&options)
		if err != nil {
			return nil, fmt.Errorf("apply plain authenticator options: %w", err)
		}
	}

	return &plainAuthenticator{
		logger: logger.WithField("authenticator", authName),
		opts:   options,
	}, nil
}

func (p plainAuthenticator) Login(_ context.Context, request *apis.UserPasswordAuthRequest) (*apis.UserPasswordAuthResponse, error) {
	password := p.opts.credentials[request.Username]
	if password != "" && password == request.Password {
		return &apis.UserPasswordAuthResponse{
			ReturnCode: apis.AuthAccepted,
		}, nil
	}
	return &apis.UserPasswordAuthResponse{
		ReturnCode: apis.AuthUnauthorized,
	}, nil
}

func (p plainAuthenticator) Close() error {
	return nil
}

func (p plainAuthenticator) Name() string {
	return authName
}
