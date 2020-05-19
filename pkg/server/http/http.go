package http

import (
	"context"
	"net/http"
	"net/http/pprof"

	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/grepplabs/mqtt-proxy/pkg/prober"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	logger log.Logger
	prober *prober.HTTPProbe

	mux *http.ServeMux
	srv *http.Server

	opts options
}

// New creates a new Server.
func New(logger log.Logger, registry *prometheus.Registry, prober *prober.HTTPProbe, opts ...Option) *Server {
	options := options{}
	for _, o := range opts {
		o.apply(&options)
	}

	mux := http.NewServeMux()
	registerMetrics(mux, registry)
	registerProbes(mux, prober, logger)
	registerProfiler(mux)

	return &Server{
		logger: logger.WithField("service", "http/server"),
		prober: prober,
		mux:    mux,
		srv:    &http.Server{Addr: options.listen, Handler: mux},
		opts:   options,
	}
}

func (s *Server) ListenAndServe() error {
	s.logger.WithField("address", s.opts.listen).Infof("listening for HTTP requests and metrics")
	return errors.Wrap(s.srv.ListenAndServe(), "serve HTTP and metrics")
}

func (s *Server) Shutdown(err error) {
	if err == http.ErrServerClosed {
		s.logger.Warnf("internal server closed unexpectedly")
		return
	}

	defer s.logger.WithError(err).Infof("internal server shutdown")

	if s.opts.gracePeriod == 0 {
		_ = s.srv.Close()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.opts.gracePeriod)
	defer cancel()

	if err := s.srv.Shutdown(ctx); err != nil {
		s.logger.WithError(err).Infof("internal server shut down failed")
	}
}

func (s *Server) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

func registerProfiler(mux *http.ServeMux) {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}

func registerMetrics(mux *http.ServeMux, g prometheus.Gatherer) {
	if g != nil {
		mux.Handle("/metrics", promhttp.HandlerFor(g, promhttp.HandlerOpts{}))
	}
}

func registerProbes(mux *http.ServeMux, p *prober.HTTPProbe, logger log.Logger) {
	if p != nil {
		mux.Handle("/healthy", p.HealthyHandler(logger))
		mux.Handle("/ready", p.ReadyHandler(logger))
	}
}
