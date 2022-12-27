package mqttserver

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"runtime"
	"time"

	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"go.uber.org/atomic"

	mqttcodec "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec"
	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

const (
	defaultReaderBufferSize = 1024
	defaultWriteBufferSize  = 1024
)

func ReadMQTTMessage(reader io.Reader, protocolVersion byte) (mqttproto.ControlPacket, error) {
	return mqttcodec.ReadPacket(reader, protocolVersion)
}

// conn represents the server side of a mqtt connection.
type conn struct {
	server *Server    // the Server on which the connection arrived
	rwc    net.Conn   // i/o connection
	logger log.Logger // logger

	bufr *bufio.Reader
	bufw *bufio.Writer

	tlsState *tls.ConnectionState // or nil when not using TLS
	writer   *response            // the mqtt.Conn exposed to handlers

	curState atomic.Uint64 // packed (unixtime<<8|uint8(ConnState))

}

// Read next message from connection.
func (c *conn) readRequest(ctx context.Context, properties Properties) (w *response, req mqttproto.ControlPacket, err error) {
	if c.server.ReadTimeout > 0 {
		_ = c.rwc.SetReadDeadline(time.Now().Add(c.server.ReadTimeout))
	}
	req, err = ReadMQTTMessage(c.bufr, properties.ProtocolVersion())
	if err != nil {
		return nil, nil, err
	}
	properties.SetProtocolVersion(req.Version())
	return &response{conn: c, ctx: ctx, properties: properties}, req, nil
}

// Serve a new connection.
func (c *conn) serve(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, 4096)
			buf = buf[:runtime.Stack(buf, false)]
			c.logger.WithField("recover", fmt.Sprintf("%v", err)).WithField("stack", fmt.Sprintf("%s", buf)).Errorf("mqtt: panic serving from /%v", c.rwc.RemoteAddr())
		}
		_ = c.rwc.Close()
		c.setState(StateClosed)
	}()
	if tlsConn, ok := c.rwc.(*tls.Conn); ok {
		if d := c.server.ReadTimeout; d != 0 {
			_ = c.rwc.SetReadDeadline(time.Now().Add(d))
		}
		if d := c.server.WriteTimeout; d != 0 {
			_ = c.rwc.SetWriteDeadline(time.Now().Add(d))
		}
		if err := tlsConn.Handshake(); err != nil {
			//TODO: log error in debug mode
			return
		}
		c.tlsState = &tls.ConnectionState{}
		*c.tlsState = tlsConn.ConnectionState()
	}

	c.bufr = bufio.NewReaderSize(c.rwc, getBufferSize(c.server.ReaderBufferSize, defaultReaderBufferSize))
	c.bufw = bufio.NewWriterSize(c.rwc, getBufferSize(c.server.WriterBufferSize, defaultWriteBufferSize))

	properties := &properties{}
	// default idle timeout - can be overridden be KeepAlive from the CONN packet
	properties.SetIdleTimeout(c.server.IdleTimeout)

	for {
		w, req, err := c.readRequest(ctx, properties)

		c.setState(StateActive)

		if err != nil {
			if rp, ok := err.(mqttproto.ResponsePacket); ok {
				_ = rp.Response().Write(c.rwc)
			}
			_ = c.rwc.Close()
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				// can report error to some channel
			}
			break
		}
		serverHandler{c.server}.ServeMQTT(w, req)

		c.setState(StateIdle)

		if !w.conn.server.doKeepAlives() {
			// we're in shutdown mode
			return
		}
		if d := w.Properties().IdleTimeout(); d != 0 {
			_ = c.rwc.SetReadDeadline(time.Now().Add(d))
			// 2 bytes = mqtt fixed header + length
			if _, err := c.bufr.Peek(2); err != nil {
				return
			}
		}
		_ = c.rwc.SetReadDeadline(time.Time{})
	}
}

func (c *conn) getState() (state ConnState, unixSec int64) {
	packedState := c.curState.Load()
	return ConnState(packedState & 0xff), int64(packedState >> 8)
}

func (c *conn) setState(state ConnState) {
	srv := c.server
	switch state {
	case StateNew:
		srv.trackConn(c, true)
	case StateClosed:
		srv.trackConn(c, false)
	}
	if state > 0xff || state < 0 {
		panic("internal error")
	}
	packedState := uint64(time.Now().Unix()<<8) | uint64(state)
	c.curState.Store(packedState)
}

type ConnState int32

const (
	StateNew ConnState = iota
	StateActive
	StateIdle
	StateClosed
)

var stateName = map[ConnState]string{
	StateNew:    "new",
	StateActive: "active",
	StateIdle:   "idle",
	StateClosed: "closed",
}

func (c ConnState) String() string {
	return stateName[c]
}

func getBufferSize(size, defaultSize int) int {
	if size > 0 {
		return size
	}
	return defaultSize
}
