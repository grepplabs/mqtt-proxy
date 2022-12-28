package v5

import (
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

type PingreqPacket struct {
	mqttproto.FixedHeader
}

func (p *PingreqPacket) Type() byte {
	return p.MessageType
}

func (p *PingreqPacket) Version() byte {
	return mqttproto.MQTT_5
}

func (p *PingreqPacket) Name() string {
	return "PINGREQ"
}

func (p *PingreqPacket) String() string {
	return p.FixedHeader.String()
}

func (p *PingreqPacket) Write(w io.Writer) (err error) {
	packet := p.FixedHeader.Pack()
	_, err = packet.WriteTo(w)
	return err
}

func (p *PingreqPacket) Unpack(_ io.Reader) error {
	return nil
}
