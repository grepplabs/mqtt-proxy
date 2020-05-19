package mqttcodec

import (
	"fmt"
	"io"
)

type ConnackPacket struct {
	FixedHeader
	SessionPresent bool
	ReturnCode     byte
}

func (p *ConnackPacket) Type() byte {
	return p.MessageType
}

func (p *ConnackPacket) Name() string {
	return "CONNACK"
}

func (p *ConnackPacket) String() string {
	return fmt.Sprintf("%s SessionPresent: %t ReturnCode: %d", p.FixedHeader, p.SessionPresent, p.ReturnCode)
}

func (p *ConnackPacket) Write(w io.Writer) (err error) {
	p.FixedHeader.RemainingLength = 2
	packet := p.FixedHeader.pack()
	packet.WriteByte(p.getConnAckFlags())
	packet.WriteByte(p.ReturnCode)
	_, err = packet.WriteTo(w)
	return err
}

func (p *ConnackPacket) Unpack(b io.Reader) error {
	connAckFlags, err := decodeByte(b)
	if err != nil {
		return err
	}
	p.SessionPresent = (connAckFlags & 0x01) == 0x01
	p.ReturnCode, err = decodeByte(b)
	return err
}

func (p *ConnackPacket) getConnAckFlags() byte {
	if p.SessionPresent {
		return 0x01
	}
	return 0x00
}
