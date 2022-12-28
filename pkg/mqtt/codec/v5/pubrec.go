package v5

import (
	"bytes"
	"fmt"
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

type PubrecPacket struct {
	mqttproto.FixedHeader
	MessageID        uint16
	ReasonCode       byte
	PubrecProperties Properties
}

func (p *PubrecPacket) Type() byte {
	return p.MessageType
}

func (p *PubrecPacket) Version() byte {
	return mqttproto.MQTT_5
}

func (p *PubrecPacket) Name() string {
	return "PUBREC"
}

func (p *PubrecPacket) String() string {
	return fmt.Sprintf("%v MessageID: %d ReasonCode: %d", p.FixedHeader, p.MessageID, p.ReasonCode)
}

func (p *PubrecPacket) Write(w io.Writer) (err error) {
	if p.ReasonCode == 0 && len(p.PubrecProperties.RawData) == 0 {
		p.FixedHeader.RemainingLength = 2
		packet := p.FixedHeader.Pack()
		packet.Write(mqttproto.EncodeUint16(p.MessageID))
		_, err = packet.WriteTo(w)
		return err
	} else {
		var body bytes.Buffer
		body.Write(mqttproto.EncodeUint16(p.MessageID))
		body.WriteByte(p.ReasonCode)
		body.Write(p.PubrecProperties.Encode())

		p.FixedHeader.RemainingLength = body.Len()
		packet := p.FixedHeader.Pack()
		packet.Write(body.Bytes())
		_, err = packet.WriteTo(w)
		return err
	}
}

func (p *PubrecPacket) Unpack(b io.Reader) (err error) {
	p.MessageID, err = mqttproto.DecodeUint16(b)
	if err != nil {
		return err
	}
	// 3.5.2.1 PUBREC Reason Code
	if p.RemainingLength == 2 {
		p.ReasonCode = 0
		p.PubrecProperties = Properties{RawData: []byte{}}
	} else {
		p.ReasonCode, err = mqttproto.DecodeByte(b)
		if err != nil {
			return err
		}
		err = p.PubrecProperties.Unpack(b)
		if err != nil {
			return err
		}
	}
	return err
}
