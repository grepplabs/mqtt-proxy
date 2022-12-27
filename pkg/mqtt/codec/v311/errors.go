package v311

import (
	"fmt"
	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

func NewConnAckError(returnCode byte, text string) error {
	return &ConnectAckError{rc: returnCode, s: text}
}

type ConnectAckError struct {
	rc byte
	s  string
}

func (e *ConnectAckError) Error() string {
	return fmt.Sprintf("CONNACK return code %d: %s", e.rc, e.s)
}

func (e *ConnectAckError) ReturnCode() byte {
	return e.rc
}

func (e *ConnectAckError) Response() mqttproto.ControlPacket {
	packet := NewControlPacket(mqttproto.CONNACK).(*ConnackPacket)
	packet.ReturnCode = e.rc
	return packet
}
