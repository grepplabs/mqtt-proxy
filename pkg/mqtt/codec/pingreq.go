package mqttcodec

import (
	"io"
)

type PingreqPacket struct {
	FixedHeader
}

func (p *PingreqPacket) Type() byte {
	return p.MessageType
}

func (p *PingreqPacket) Name() string {
	return "PINGREQ"
}

func (p *PingreqPacket) String() string {
	return p.FixedHeader.String()
}

func (p *PingreqPacket) Write(w io.Writer) (err error) {
	packet := p.FixedHeader.pack()
	_, err = packet.WriteTo(w)
	return err
}

func (p *PingreqPacket) Unpack(_ io.Reader) error {
	return nil
}
