package v5

import (
	"bytes"
	"fmt"
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

type DisconnectPacket struct {
	mqttproto.FixedHeader
	ReasonCode           byte
	DisconnectProperties Properties
}

func (p *DisconnectPacket) Type() byte {
	return p.MessageType
}

func (p *DisconnectPacket) Version() byte {
	return mqttproto.MQTT_5
}

func (p *DisconnectPacket) Name() string {
	return "DISCONNECT"
}

func (p *DisconnectPacket) String() string {
	return fmt.Sprintf("%v ReasonCode: %d", p.FixedHeader, p.ReasonCode)
}

func (p *DisconnectPacket) Write(w io.Writer) (err error) {
	if p.ReasonCode == 0 && len(p.DisconnectProperties.RawData) == 0 {
		packet := p.FixedHeader.Pack()
		_, err = packet.WriteTo(w)
		return err
	} else {
		var body bytes.Buffer
		body.WriteByte(p.ReasonCode)
		body.Write(p.DisconnectProperties.Encode())

		p.FixedHeader.RemainingLength = body.Len()
		packet := p.FixedHeader.Pack()
		packet.Write(body.Bytes())
		_, err = packet.WriteTo(w)
		return err
	}
}

func (p *DisconnectPacket) Unpack(b io.Reader) (err error) {
	// 3.14.2.1 Disconnect Reason Code
	if p.RemainingLength == 0 {
		p.ReasonCode = 0
		p.DisconnectProperties = Properties{RawData: []byte{}}
	} else {
		p.ReasonCode, err = mqttproto.DecodeByte(b)
		if err != nil {
			return err
		}
		err = p.DisconnectProperties.Unpack(b)
		if err != nil {
			return err
		}
	}
	return err
}
