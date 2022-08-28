package mqttserver

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"time"

	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

var (
	ErrServerClosed      = errors.New("mqtt: Server closed")
	shutdownPollInterval = 500 * time.Millisecond
)

// A Server defines parameters for running a mqtt server.
type Server struct {
	Network      string        // network of the address - empty string defaults to tcp
	Addr         string        // address to listen on, ":1883" if empty
	Handler      Handler       // handler to invoke, DefaultServeMux if nil
	ReadTimeout  time.Duration // maximum duration before timing out read of the request
	WriteTimeout time.Duration // maximum duration before timing out write of the response
	IdleTimeout  time.Duration // maximum amount of time to wait for the next request
	TLSConfig    *tls.Config   // optional TLS config, used by ListenAndServeTLS

	ReaderBufferSize int // read buffer size pro tcp connection (default 1024)
	WriterBufferSize int // write buffer size pro tcp connection (default 1024)

	ErrorLog log.Logger

	ConnDebug func(c net.Conn) net.Conn // optional logging wrapper for all server connections

	inShutdown atomic.Bool // true when when server is in shutdown

	mu         sync.Mutex
	listeners  map[*net.Listener]struct{}
	activeConn map[*conn]struct{}
	doneChan   chan struct{}

	totalConn int64 // metric counting total number of connections
}

func (srv *Server) Serve(l net.Listener) (err error) {
	l = &onceCloseListener{Listener: l}
	defer l.Close()

	if !srv.trackListener(&l, true) {
		return ErrServerClosed
	}
	defer srv.trackListener(&l, false)

	var tempDelay time.Duration // how long to sleep on accept failure

	for {
		rw, err := l.Accept()
		if err != nil {
			select {
			case <-srv.getDoneChan():
				return ErrServerClosed
			default:
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				srv.logf("mqtt: accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		tempDelay = 0
		c := srv.newConn(rw)
		c.setState(StateNew) // before Serve can return
		go c.serve(context.Background())
	}
}

// Create new connection from rwc.
func (srv *Server) newConn(rwc net.Conn) *conn {
	logger := srv.ErrorLog
	if logger == nil {
		logger = log.GetInstance()
	}
	c := &conn{
		server: srv,
		rwc:    rwc,
		logger: logger,
	}
	if connDebug := srv.ConnDebug; connDebug != nil {
		c.rwc = connDebug(c.rwc)
		if c.rwc == nil {
			panic("ConnDebug returned nil")
		}
	}
	return c
}

func (srv *Server) ListenAndServe() (err error) {
	if srv.inShutdown.Load() {
		return ErrServerClosed
	}
	network := srv.Network
	if len(network) == 0 {
		network = "tcp"
	}
	addr := srv.Addr
	if len(addr) == 0 {
		addr = ":1883"
	}
	var conn net.Listener
	conn, err = net.Listen(network, addr)
	if err != nil {
		return err
	}

	l := &onceCloseListener{Listener: conn}
	defer l.Close()

	return srv.Serve(l)
}

func (srv *Server) ListenAndServeTLS(certFile, keyFile string) (err error) {
	if srv.inShutdown.Load() {
		return ErrServerClosed
	}
	network := srv.Network
	if len(network) == 0 {
		network = "tcp"
	}
	addr := srv.Addr
	if len(addr) == 0 {
		addr = ":8883"
	}
	var conn net.Listener
	conn, err = net.Listen(network, addr)
	if err != nil {
		return err
	}

	l := &onceCloseListener{Listener: conn}
	defer l.Close()

	return srv.ServeTLS(l, certFile, keyFile)
}

func (srv *Server) ServeTLS(l net.Listener, certFile, keyFile string) error {
	config := srv.TLSConfig

	configHasCert := len(config.Certificates) > 0 || config.GetCertificate != nil
	if !configHasCert || certFile != "" || keyFile != "" {
		var err error
		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return err
		}
	}

	tlsListener := tls.NewListener(l, config)
	return srv.Serve(tlsListener)
}

func (srv *Server) Close() error {
	srv.inShutdown.Store(true)
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.closeDoneChanLocked()

	err := srv.closeListenersLocked()
	for c := range srv.activeConn {
		_ = c.rwc.Close()
		delete(srv.activeConn, c)
	}
	return err
}

func (srv *Server) Shutdown(ctx context.Context) error {
	srv.inShutdown.Store(true)

	srv.mu.Lock()
	lnerr := srv.closeListenersLocked()
	srv.closeDoneChanLocked()
	srv.mu.Unlock()

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if srv.closeIdleConns() && srv.numListeners() == 0 {
			return lnerr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (srv *Server) closeIdleConns() bool {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	quiescent := true
	for c := range srv.activeConn {
		st, unixSec := c.getState()
		// treat StateNew connections as if they're idle if we haven't read the first request's header in over 5 seconds.
		if st == StateNew && unixSec < time.Now().Unix()-5 {
			st = StateIdle
		}
		if st != StateIdle || unixSec == 0 {
			// Assume unixSec == 0 means it's a very new connection, without state set yet.
			quiescent = false
			continue
		}
		_ = c.rwc.Close()
		delete(srv.activeConn, c)
	}
	return quiescent
}

func (srv *Server) closeListenersLocked() error {
	var err error
	for ln := range srv.listeners {
		if cerr := (*ln).Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}

func (srv *Server) getDoneChan() <-chan struct{} {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return srv.getDoneChanLocked()
}

func (srv *Server) getDoneChanLocked() chan struct{} {
	if srv.doneChan == nil {
		srv.doneChan = make(chan struct{})
	}
	return srv.doneChan
}

func (srv *Server) closeDoneChanLocked() {
	ch := srv.getDoneChanLocked()
	select {
	case <-ch:
		// Already closed. Don't close again.
	default:
		// Safe to close here. We're the only closer, guarded
		// by s.mu.
		close(ch)
	}
}

func (srv *Server) logf(format string, args ...interface{}) {
	if srv.ErrorLog != nil {
		srv.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

func (srv *Server) shuttingDown() bool {
	return srv.inShutdown.Load()
}

func (srv *Server) trackListener(ln *net.Listener, add bool) bool {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.listeners == nil {
		srv.listeners = make(map[*net.Listener]struct{})
	}
	if add {
		if srv.shuttingDown() {
			return false
		}
		srv.listeners[ln] = struct{}{}
	} else {
		delete(srv.listeners, ln)
	}
	return true
}

func (srv *Server) trackConn(c *conn, add bool) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.activeConn == nil {
		srv.activeConn = make(map[*conn]struct{})
	}
	if add {
		srv.totalConn++
		srv.activeConn[c] = struct{}{}
	} else {
		delete(srv.activeConn, c)
	}
}

func (srv *Server) doKeepAlives() bool {
	return !srv.shuttingDown()
}

func (srv *Server) numListeners() int {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return len(srv.listeners)
}

// Number of active connections
func (srv *Server) NumActiveConn() int {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return len(srv.activeConn)
}

// Number of total connections
func (srv *Server) NumTotalConn() int64 {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return srv.totalConn
}

func ListenAndServe(addr string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler, ErrorLog: log.GetInstance()}
	return server.ListenAndServe()
}

func ListenAndServeTLS(addr, certFile, keyFile string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler, ErrorLog: log.GetInstance()}
	return server.ListenAndServeTLS(certFile, keyFile)
}

// onceCloseListener wraps a net.Listener, protecting it from  multiple Close calls.
type onceCloseListener struct {
	net.Listener
	once     sync.Once
	closeErr error
}

func (oc *onceCloseListener) Close() error {
	oc.once.Do(oc.close)
	return oc.closeErr
}

func (oc *onceCloseListener) close() { oc.closeErr = oc.Listener.Close() }
