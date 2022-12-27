package v311

import (
	"fmt"
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

type UnsubackPacket struct {
	mqttproto.FixedHeader
	MessageID uint16
}

func (p *UnsubackPacket) Type() byte {
	return p.MessageType
}

func (p *UnsubackPacket) Version() byte {
	return mqttproto.MQTT_3_1_1
}

func (p *UnsubackPacket) Name() string {
	return "UNSUBACK"
}

func (p *UnsubackPacket) String() string {
	return fmt.Sprintf("%v MessageID: %d", p.FixedHeader, p.MessageID)
}

func (p *UnsubackPacket) Write(w io.Writer) (err error) {
	p.FixedHeader.RemainingLength = 2
	packet := p.FixedHeader.Pack()
	packet.Write(mqttproto.EncodeUint16(p.MessageID))
	_, err = packet.WriteTo(w)
	return err
}

func (p *UnsubackPacket) Unpack(b io.Reader) (err error) {
	p.MessageID, err = mqttproto.DecodeUint16(b)
	return err
}
