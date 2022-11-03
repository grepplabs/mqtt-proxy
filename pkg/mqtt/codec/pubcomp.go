package mqttcodec

import (
	"fmt"
	"io"
)

type PubcompPacket struct {
	FixedHeader
	MessageID uint16
}

func (p *PubcompPacket) Type() byte {
	return p.MessageType
}

func (p *PubcompPacket) Name() string {
	return "PUBCOMP"
}

func (p *PubcompPacket) String() string {
	return fmt.Sprintf("%v MessageID: %d", p.FixedHeader, p.MessageID)
}

func (p *PubcompPacket) Write(w io.Writer) (err error) {
	p.FixedHeader.RemainingLength = 2
	packet := p.FixedHeader.pack()
	packet.Write(encodeUint16(p.MessageID))
	_, err = packet.WriteTo(w)
	return err
}

func (p *PubcompPacket) Unpack(b io.Reader) (err error) {
	p.MessageID, err = decodeUint16(b)
	return err
}
