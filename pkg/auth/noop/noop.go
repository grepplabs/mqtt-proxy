package noop

import (
	"context"

	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	authName = "noop"
)

type noopAuthenticator struct {
	logger log.Logger
}

func New(logger log.Logger, _ *prometheus.Registry) apis.UserPasswordAuthenticator {
	return &noopAuthenticator{
		logger: logger.WithField("authenticator", authName),
	}
}

func (p noopAuthenticator) Login(_ context.Context, _ *apis.UserPasswordAuthRequest) (*apis.UserPasswordAuthResponse, error) {
	return &apis.UserPasswordAuthResponse{
		ReturnCode: apis.AuthAccepted,
	}, nil
}

func (p noopAuthenticator) Close() error {
	return nil
}

func (p noopAuthenticator) Name() string {
	return authName
}
