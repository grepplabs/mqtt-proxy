package v311

import (
	"fmt"
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

type PubackPacket struct {
	mqttproto.FixedHeader
	MessageID uint16
}

func (p *PubackPacket) Type() byte {
	return p.MessageType
}

func (p *PubackPacket) Version() byte {
	return mqttproto.MQTT_3_1_1
}

func (p *PubackPacket) Name() string {
	return "PUBACK"
}

func (p *PubackPacket) String() string {
	return fmt.Sprintf("%v MessageID: %d", p.FixedHeader, p.MessageID)
}

func (p *PubackPacket) Write(w io.Writer) (err error) {
	p.FixedHeader.RemainingLength = 2
	packet := p.FixedHeader.Pack()
	packet.Write(mqttproto.EncodeUint16(p.MessageID))
	_, err = packet.WriteTo(w)
	return err
}

func (p *PubackPacket) Unpack(b io.Reader) (err error) {
	p.MessageID, err = mqttproto.DecodeUint16(b)
	return err
}
