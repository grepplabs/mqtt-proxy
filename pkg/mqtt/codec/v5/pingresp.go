package v5

import (
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

type PingrespPacket struct {
	mqttproto.FixedHeader
}

func (p *PingrespPacket) Type() byte {
	return p.MessageType
}

func (p *PingrespPacket) Version() byte {
	return mqttproto.MQTT_5
}

func (p *PingrespPacket) Name() string {
	return "PINGRESP"
}

func (p *PingrespPacket) String() string {
	return p.FixedHeader.String()
}

func (p *PingrespPacket) Write(w io.Writer) (err error) {
	packet := p.FixedHeader.Pack()
	_, err = packet.WriteTo(w)
	return err
}

func (p *PingrespPacket) Unpack(_ io.Reader) error {
	return nil
}
