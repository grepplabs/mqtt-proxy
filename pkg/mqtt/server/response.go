package mqttserver

import (
	"context"
	"crypto/tls"
	"go.uber.org/atomic"
	"io"
	"net"
	"sync"
	"time"
)

type Properties interface {
	IdleTimeout() time.Duration   // Returns connection idle timeout
	SetIdleTimeout(time.Duration) // Store a new idle timeout

	Authenticated() bool   // Returns if the connection was authenticated
	SetAuthenticated(bool) // Store the authenticated flag
}

type properties struct {
	idleTimeout   atomic.Duration
	authenticated atomic.Bool
}

func (w *properties) IdleTimeout() time.Duration {
	return w.idleTimeout.Load()
}

func (w *properties) SetIdleTimeout(d time.Duration) {
	w.idleTimeout.Store(d)
}

func (w *properties) Authenticated() bool {
	return w.authenticated.Load()
}

func (w *properties) SetAuthenticated(b bool) {
	w.authenticated.Store(b)
}

// Conn interface is used by a handler to send mqtt messages.
type Conn interface {
	io.WriteCloser
	LocalAddr() net.Addr       // Returns the local IP
	RemoteAddr() net.Addr      // Returns the remote IP
	TLS() *tls.ConnectionState // TLS or nil when not using TLS
	Context() context.Context  // Returns the internal context
	Connection() net.Conn      // Returns network connection
	Properties() Properties    // Data set by handler during connection duration
}

// A response represents the server side of a mqtt response.
// It implements the Conn and CloseNotifier interfaces.
type response struct {
	mu         sync.Mutex      // guards conn and Write
	conn       *conn           // socket, reader and writer
	ctx        context.Context // context for this Conn
	properties Properties      // properties for this Conn
}

// Write writes the message m to the connection.
func (w *response) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn.server.WriteTimeout > 0 {
		_ = w.conn.rwc.SetWriteDeadline(time.Now().Add(w.conn.server.WriteTimeout))
	}
	n, err := w.conn.bufw.Write(b)
	if err != nil {
		return 0, err
	}
	if err = w.conn.bufw.Flush(); err != nil {
		return 0, err
	}
	return n, nil
}

// Close closes the connection.
func (w *response) Close() error {
	return w.conn.rwc.Close()
}

// LocalAddr returns the local address of the connection.
func (w *response) LocalAddr() net.Addr {
	return w.conn.rwc.LocalAddr()
}

// RemoteAddr returns the peer address of the connection.
func (w *response) RemoteAddr() net.Addr {
	return w.conn.rwc.RemoteAddr()
}

// TLS returns the TLS connection state, or nil.
func (w *response) TLS() *tls.ConnectionState {
	return w.conn.tlsState
}

// Context returns the internal context or a new context.Background.
func (w *response) Context() context.Context {
	return w.ctx
}

func (w *response) Connection() net.Conn {
	return w.conn.rwc
}

func (w *response) Properties() Properties {
	return w.properties
}
