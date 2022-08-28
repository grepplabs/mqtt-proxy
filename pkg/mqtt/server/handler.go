package mqttserver

import (
	"sync"

	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec"
)

type Handler interface {
	ServeMQTT(Conn, mqttcodec.ControlPacket)
}

type HandlerFunc func(Conn, mqttcodec.ControlPacket)

func (f HandlerFunc) ServeMQTT(c Conn, req mqttcodec.ControlPacket) {
	f(c, req)
}

type serverHandler struct {
	srv *Server
}

func (sh serverHandler) ServeMQTT(w Conn, req mqttcodec.ControlPacket) {
	handler := sh.srv.Handler
	if handler == nil {
		handler = DefaultServeMux
	}
	handler.ServeMQTT(w, req)
}

type muxEntry struct {
	h           Handler
	messageType byte
}

type ServeMux struct {
	mu     sync.RWMutex // Guards m.
	m      map[byte]muxEntry
	logger log.Logger
}

func NewServeMux(logger log.Logger) *ServeMux {
	return &ServeMux{
		m:      make(map[byte]muxEntry),
		logger: logger,
	}
}

var DefaultServeMux = NewServeMux(log.GetInstance())

func (mux *ServeMux) Handle(messageType byte, handler Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	if handler == nil {
		panic("MQTT: nil handler")
	}
	mux.m[messageType] = muxEntry{h: handler, messageType: messageType}
}

func (mux *ServeMux) ServeMQTT(c Conn, p mqttcodec.ControlPacket) {
	entry := mux.m[p.Type()]
	if entry.h == nil {
		mux.logger.Warnf("No handler available for MQTT message '%s' from /%v. Disconnecting", p.Name(), c.RemoteAddr())
		_ = c.Close()
		return
	}
	entry.h.ServeMQTT(c, p)
}
