package mqttcodec

import (
	"fmt"
	"io"
)

type UnsubackPacket struct {
	FixedHeader
	MessageID uint16
}

func (p *UnsubackPacket) Type() byte {
	return p.MessageType
}

func (p *UnsubackPacket) Name() string {
	return "UNSUBACK"
}

func (p *UnsubackPacket) String() string {
	return fmt.Sprintf("%v MessageID: %d", p.FixedHeader, p.MessageID)
}

func (p *UnsubackPacket) Write(w io.Writer) (err error) {
	p.FixedHeader.RemainingLength = 2
	packet := p.FixedHeader.pack()
	packet.Write(encodeUint16(p.MessageID))
	_, err = packet.WriteTo(w)
	return err
}

func (p *UnsubackPacket) Unpack(b io.Reader) (err error) {
	p.MessageID, err = decodeUint16(b)
	return err
}
