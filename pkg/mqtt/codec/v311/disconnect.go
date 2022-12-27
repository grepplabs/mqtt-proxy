package v311

import (
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

type DisconnectPacket struct {
	mqttproto.FixedHeader
}

func (p *DisconnectPacket) Type() byte {
	return p.MessageType
}

func (p *DisconnectPacket) Version() byte {
	return mqttproto.MQTT_3_1_1
}

func (p *DisconnectPacket) Name() string {
	return "DISCONNECT"
}

func (p *DisconnectPacket) String() string {
	return p.FixedHeader.String()
}

func (p *DisconnectPacket) Write(w io.Writer) (err error) {
	packet := p.FixedHeader.Pack()
	_, err = packet.WriteTo(w)
	return err
}

func (p *DisconnectPacket) Unpack(_ io.Reader) error {
	return nil
}
