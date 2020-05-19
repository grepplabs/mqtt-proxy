package mqtt

import (
	"context"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	mqttserver "github.com/grepplabs/mqtt-proxy/pkg/mqtt/server"
	"github.com/grepplabs/mqtt-proxy/pkg/prober"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

type Server struct {
	logger log.Logger
	prober *prober.HTTPProbe

	mux *mqttserver.ServeMux
	srv *mqttserver.Server

	opts options
}

// New creates a new Server.
func New(logger log.Logger, registry *prometheus.Registry, prober *prober.HTTPProbe, opts ...Option) *Server {
	options := options{}
	for _, o := range opts {
		o.apply(&options)
	}
	mux := mqttserver.NewServeMux(logger)

	s := &mqttserver.Server{
		Network:      options.network,
		Addr:         options.listen,
		Handler:      options.handler,
		ReadTimeout:  options.readTimeout,
		WriteTimeout: options.writeTimeout,
		TLSConfig:    options.tlsConfig,
		ErrorLog:     logger,
	}

	return &Server{
		logger: logger.WithField("service", "mqtt/server"),
		prober: prober,
		mux:    mux,
		srv:    s,
		opts:   options,
	}
}

func (s *Server) ActiveConnections() int {
	return s.srv.NumActiveConn()
}

func (s *Server) TotalConnections() int64 {
	return s.srv.NumTotalConn()
}

func (s *Server) ListenAndServe() error {
	if s.opts.tlsConfig == nil {
		s.logger.WithField("address", s.opts.listen).Infof("listening for MQTT requests")
		return errors.Wrap(s.srv.ListenAndServe(), "serve MQTT")
	} else {
		s.logger.WithField("address", s.opts.listen).Infof("listening TLS for MQTT request")
		return errors.Wrap(s.srv.ListenAndServeTLS("", ""), "serve TLS MQTT")
	}
}

// Shutdown gracefully shuts down the server by waiting
// for specified amount of time (by gracePeriod)
// for connections to return to idle and then shut down.
func (s *Server) Shutdown(err error) {
	defer s.logger.WithError(err).Infof("internal server shutdown")
	if s.opts.gracePeriod == 0 {
		_ = s.srv.Close()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.opts.gracePeriod)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		s.logger.Infof("gracefully stopping internal server")
		_ = s.srv.Shutdown(ctx)
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		s.logger.Infof("grace period exceeded enforcing shutdown")
		_ = s.srv.Close()
	case <-stopped:
		cancel()
	}
}
