package v5

import (
	"bytes"
	"fmt"
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

type PubrelPacket struct {
	mqttproto.FixedHeader
	MessageID        uint16
	ReasonCode       byte
	PubrelProperties Properties
}

func (p *PubrelPacket) Type() byte {
	return p.MessageType
}

func (p *PubrelPacket) Version() byte {
	return mqttproto.MQTT_5
}

func (p *PubrelPacket) Name() string {
	return "PUBREL"
}

func (p *PubrelPacket) String() string {
	return fmt.Sprintf("%v MessageID: %d ReasonCode: %d", p.FixedHeader, p.MessageID, p.ReasonCode)
}

func (p *PubrelPacket) Write(w io.Writer) (err error) {
	if p.ReasonCode == 0 && len(p.PubrelProperties.RawData) == 0 {
		p.FixedHeader.RemainingLength = 2
		packet := p.FixedHeader.Pack()
		packet.Write(mqttproto.EncodeUint16(p.MessageID))
		_, err = packet.WriteTo(w)
		return err
	} else {
		var body bytes.Buffer
		body.Write(mqttproto.EncodeUint16(p.MessageID))
		body.WriteByte(p.ReasonCode)
		body.Write(p.PubrelProperties.Encode())

		p.FixedHeader.RemainingLength = body.Len()
		packet := p.FixedHeader.Pack()
		packet.Write(body.Bytes())
		_, err = packet.WriteTo(w)
		return err
	}
}

func (p *PubrelPacket) Unpack(b io.Reader) (err error) {
	p.MessageID, err = mqttproto.DecodeUint16(b)
	if err != nil {
		return err
	}
	// 3.6.2.1 PUBREL Reason Code
	if p.RemainingLength == 2 {
		p.ReasonCode = 0
		p.PubrelProperties = Properties{RawData: []byte{}}
	} else {
		p.ReasonCode, err = mqttproto.DecodeByte(b)
		if err != nil {
			return err
		}
		err = p.PubrelProperties.Unpack(b)
		if err != nil {
			return err
		}
	}
	return err
}
