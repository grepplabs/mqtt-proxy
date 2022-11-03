package mqttcodec

import (
	"fmt"
	"io"
)

type PubackPacket struct {
	FixedHeader
	MessageID uint16
}

func (p *PubackPacket) Type() byte {
	return p.MessageType
}

func (p *PubackPacket) Name() string {
	return "PUBACK"
}

func (p *PubackPacket) String() string {
	return fmt.Sprintf("%v MessageID: %d", p.FixedHeader, p.MessageID)
}

func (p *PubackPacket) Write(w io.Writer) (err error) {
	p.FixedHeader.RemainingLength = 2
	packet := p.FixedHeader.pack()
	packet.Write(encodeUint16(p.MessageID))
	_, err = packet.WriteTo(w)
	return err
}

func (p *PubackPacket) Unpack(b io.Reader) (err error) {
	p.MessageID, err = decodeUint16(b)
	return err
}
