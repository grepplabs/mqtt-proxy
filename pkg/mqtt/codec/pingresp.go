package mqttcodec

import (
	"io"
)

type PingrespPacket struct {
	FixedHeader
}

func (p *PingrespPacket) Type() byte {
	return p.MessageType
}

func (p *PingrespPacket) Name() string {
	return "PINGRESP"
}

func (p *PingrespPacket) String() string {
	return p.FixedHeader.String()
}

func (p *PingrespPacket) Write(w io.Writer) (err error) {
	packet := p.FixedHeader.pack()
	_, err = packet.WriteTo(w)
	return err
}

func (p *PingrespPacket) Unpack(_ io.Reader) error {
	return nil
}
