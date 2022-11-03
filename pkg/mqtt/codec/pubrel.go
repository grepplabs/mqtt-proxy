package mqttcodec

import (
	"fmt"
	"io"
)

type PubrelPacket struct {
	FixedHeader
	MessageID uint16
}

func (p *PubrelPacket) Type() byte {
	return p.MessageType
}

func (p *PubrelPacket) Name() string {
	return "PUBREL"
}

func (p *PubrelPacket) String() string {
	return fmt.Sprintf("%v MessageID: %d", p.FixedHeader, p.MessageID)
}

func (p *PubrelPacket) Write(w io.Writer) (err error) {
	p.FixedHeader.RemainingLength = 2
	packet := p.FixedHeader.pack()
	packet.Write(encodeUint16(p.MessageID))
	_, err = packet.WriteTo(w)
	return err
}

func (p *PubrelPacket) Unpack(b io.Reader) (err error) {
	p.MessageID, err = decodeUint16(b)
	return err
}
