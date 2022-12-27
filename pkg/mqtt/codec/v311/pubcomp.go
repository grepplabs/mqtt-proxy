package v311

import (
	"fmt"
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

type PubcompPacket struct {
	mqttproto.FixedHeader
	MessageID uint16
}

func (p *PubcompPacket) Type() byte {
	return p.MessageType
}

func (p *PubcompPacket) Version() byte {
	return mqttproto.MQTT_3_1_1
}

func (p *PubcompPacket) Name() string {
	return "PUBCOMP"
}

func (p *PubcompPacket) String() string {
	return fmt.Sprintf("%v MessageID: %d", p.FixedHeader, p.MessageID)
}

func (p *PubcompPacket) Write(w io.Writer) (err error) {
	p.FixedHeader.RemainingLength = 2
	packet := p.FixedHeader.Pack()
	packet.Write(mqttproto.EncodeUint16(p.MessageID))
	_, err = packet.WriteTo(w)
	return err
}

func (p *PubcompPacket) Unpack(b io.Reader) (err error) {
	p.MessageID, err = mqttproto.DecodeUint16(b)
	return err
}
