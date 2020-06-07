package instrument

import (
	"context"
	"strconv"
	"time"

	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type authenticator struct {
	delegate apis.UserPasswordAuthenticator
	metrics  *authenticatorMetrics
}

type authenticatorMetrics struct {
	loginDuration *prometheus.HistogramVec
}

func New(delegate apis.UserPasswordAuthenticator, registry *prometheus.Registry) apis.UserPasswordAuthenticator {
	return &authenticator{
		delegate: delegate,
		metrics:  newAuthenticatorMetrics(delegate.Name(), registry),
	}
}

func (p *authenticator) Name() string {
	return p.delegate.Name()
}

func (p *authenticator) Login(ctx context.Context, authRequest *apis.UserPasswordAuthRequest) (*apis.UserPasswordAuthResponse, error) {
	start := time.Now()
	authResponse, err := p.delegate.Login(ctx, authRequest)
	code := ""
	if authResponse != nil {
		code = strconv.Itoa(int(authResponse.ReturnCode))
	}
	isError := "0"
	if err != nil {
		isError = "1"
	}
	p.metrics.loginDuration.WithLabelValues(code, isError).Observe(time.Since(start).Seconds())
	return authResponse, err
}

func (p *authenticator) Close() error {
	return p.delegate.Close()
}

func newAuthenticatorMetrics(name string, registry *prometheus.Registry) *authenticatorMetrics {
	loginDuration := promauto.With(registry).NewHistogramVec(prometheus.HistogramOpts{
		Name:        "mqtt_proxy_authenticator_login_duration_seconds",
		Help:        "Tracks the latencies for auth requests.",
		ConstLabels: prometheus.Labels{"name": name},
	}, []string{"code", "error"})

	return &authenticatorMetrics{
		loginDuration: loginDuration,
	}
}
