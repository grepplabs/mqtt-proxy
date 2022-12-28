package v5

import (
	"bytes"
	"fmt"
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

type ConnackPacket struct {
	mqttproto.FixedHeader
	SessionPresent    bool
	ReturnCode        byte
	ConnackProperties Properties
}

func (p *ConnackPacket) Type() byte {
	return p.MessageType
}

func (p *ConnackPacket) Version() byte {
	return mqttproto.MQTT_5
}

func (p *ConnackPacket) Name() string {
	return "CONNACK"
}

func (p *ConnackPacket) String() string {
	return fmt.Sprintf("%v SessionPresent: %t ReturnCode: %d", p.FixedHeader, p.SessionPresent, p.ReturnCode)
}

func (p *ConnackPacket) Write(w io.Writer) (err error) {
	var body bytes.Buffer
	body.WriteByte(p.getConnAckFlags())
	body.WriteByte(p.ReturnCode)
	body.Write(p.ConnackProperties.Encode())

	p.FixedHeader.RemainingLength = body.Len()
	packet := p.FixedHeader.Pack()
	packet.Write(body.Bytes())
	_, err = packet.WriteTo(w)
	return err
}

func (p *ConnackPacket) Unpack(b io.Reader) error {
	connAckFlags, err := mqttproto.DecodeByte(b)
	if err != nil {
		return err
	}
	p.SessionPresent = (connAckFlags & 0x01) == 0x01
	p.ReturnCode, err = mqttproto.DecodeByte(b)
	if err != nil {
		return err
	}
	err = p.ConnackProperties.Unpack(b)
	if err != nil {
		return err
	}
	return err
}

func (p *ConnackPacket) getConnAckFlags() byte {
	if p.SessionPresent {
		return 0x01
	}
	return 0x00
}
