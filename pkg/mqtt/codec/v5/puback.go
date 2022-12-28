package v5

import (
	"bytes"
	"fmt"
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

type PubackPacket struct {
	mqttproto.FixedHeader
	MessageID        uint16
	ReasonCode       byte
	PubackProperties Properties
}

func (p *PubackPacket) Type() byte {
	return p.MessageType
}

func (p *PubackPacket) Version() byte {
	return mqttproto.MQTT_5
}

func (p *PubackPacket) Name() string {
	return "PUBACK"
}

func (p *PubackPacket) String() string {
	return fmt.Sprintf("%v MessageID: %d ReasonCode: %d", p.FixedHeader, p.MessageID, p.ReasonCode)
}

func (p *PubackPacket) Write(w io.Writer) (err error) {
	if p.ReasonCode == 0 && len(p.PubackProperties.RawData) == 0 {
		p.FixedHeader.RemainingLength = 2
		packet := p.FixedHeader.Pack()
		packet.Write(mqttproto.EncodeUint16(p.MessageID))
		_, err = packet.WriteTo(w)
		return err
	} else {
		var body bytes.Buffer
		body.Write(mqttproto.EncodeUint16(p.MessageID))
		body.WriteByte(p.ReasonCode)
		body.Write(p.PubackProperties.Encode())

		p.FixedHeader.RemainingLength = body.Len()
		packet := p.FixedHeader.Pack()
		packet.Write(body.Bytes())
		_, err = packet.WriteTo(w)
		return err
	}
}

func (p *PubackPacket) Unpack(b io.Reader) (err error) {
	p.MessageID, err = mqttproto.DecodeUint16(b)
	if err != nil {
		return err
	}
	// 3.4.2.1 PUBACK Reason Code
	if p.RemainingLength == 2 {
		p.ReasonCode = 0
		p.PubackProperties = Properties{RawData: []byte{}}
	} else {
		p.ReasonCode, err = mqttproto.DecodeByte(b)
		if err != nil {
			return err
		}
		err = p.PubackProperties.Unpack(b)
		if err != nil {
			return err
		}
	}
	return err
}
