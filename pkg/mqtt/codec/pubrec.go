package mqttcodec

import (
	"fmt"
	"io"
)

type PubrecPacket struct {
	FixedHeader
	MessageID uint16
}

func (p *PubrecPacket) Type() byte {
	return p.MessageType
}

func (p *PubrecPacket) Name() string {
	return "PUBREC"
}

func (p *PubrecPacket) String() string {
	return fmt.Sprintf("%v MessageID: %d", p.FixedHeader, p.MessageID)
}

func (p *PubrecPacket) Write(w io.Writer) (err error) {
	p.FixedHeader.RemainingLength = 2
	packet := p.FixedHeader.pack()
	packet.Write(encodeUint16(p.MessageID))
	_, err = packet.WriteTo(w)
	return err
}

func (p *PubrecPacket) Unpack(b io.Reader) (err error) {
	p.MessageID, err = decodeUint16(b)
	return err
}
