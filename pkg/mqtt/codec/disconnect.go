package mqttcodec

import (
	"io"
)

type DisconnectPacket struct {
	FixedHeader
}

func (p *DisconnectPacket) Type() byte {
	return p.MessageType
}

func (p *DisconnectPacket) Name() string {
	return "DISCONNECT"
}

func (p *DisconnectPacket) String() string {
	return p.FixedHeader.String()
}

func (p *DisconnectPacket) Write(w io.Writer) (err error) {
	packet := p.FixedHeader.pack()
	_, err = packet.WriteTo(w)
	return err
}

func (p *DisconnectPacket) Unpack(_ io.Reader) error {
	return nil
}
