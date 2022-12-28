package v5

import (
	"bytes"
	"fmt"
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

type PubcompPacket struct {
	mqttproto.FixedHeader
	MessageID         uint16
	ReasonCode        byte
	PubcompProperties Properties
}

func (p *PubcompPacket) Type() byte {
	return p.MessageType
}

func (p *PubcompPacket) Version() byte {
	return mqttproto.MQTT_5
}

func (p *PubcompPacket) Name() string {
	return "PUBCOMP"
}

func (p *PubcompPacket) String() string {
	return fmt.Sprintf("%v MessageID: %d ReasonCode: %d", p.FixedHeader, p.MessageID, p.ReasonCode)
}

func (p *PubcompPacket) Write(w io.Writer) (err error) {
	if p.ReasonCode == 0 && len(p.PubcompProperties.RawData) == 0 {
		p.FixedHeader.RemainingLength = 2
		packet := p.FixedHeader.Pack()
		packet.Write(mqttproto.EncodeUint16(p.MessageID))
		_, err = packet.WriteTo(w)
		return err
	} else {
		var body bytes.Buffer
		body.Write(mqttproto.EncodeUint16(p.MessageID))
		body.WriteByte(p.ReasonCode)
		body.Write(p.PubcompProperties.Encode())

		p.FixedHeader.RemainingLength = body.Len()
		packet := p.FixedHeader.Pack()
		packet.Write(body.Bytes())
		_, err = packet.WriteTo(w)
		return err
	}
}

func (p *PubcompPacket) Unpack(b io.Reader) (err error) {
	p.MessageID, err = mqttproto.DecodeUint16(b)
	if err != nil {
		return err
	}
	// 3.7.2.1 PUBCOMP Reason Code
	if p.RemainingLength == 2 {
		p.ReasonCode = 0
		p.PubcompProperties = Properties{RawData: []byte{}}
	} else {
		p.ReasonCode, err = mqttproto.DecodeByte(b)
		if err != nil {
			return err
		}
		err = p.PubcompProperties.Unpack(b)
		if err != nil {
			return err
		}
	}
	return err
}
