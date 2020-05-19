package mqttcodec

import (
	"bytes"
	"fmt"
	"io"
)

type SubackPacket struct {
	FixedHeader
	MessageID   uint16
	ReturnCodes []byte
}

func (p *SubackPacket) Type() byte {
	return p.MessageType
}

func (p *SubackPacket) Name() string {
	return "SUBACK"
}

func (p *SubackPacket) String() string {
	return fmt.Sprintf("%s MessageID: %d ReturnCodes %v", p.FixedHeader, p.MessageID, p.ReturnCodes)
}

func (p *SubackPacket) Write(w io.Writer) (err error) {
	var body bytes.Buffer

	body.Write(encodeUint16(p.MessageID))
	body.Write(p.ReturnCodes)

	p.FixedHeader.RemainingLength = body.Len()
	packet := p.FixedHeader.pack()
	packet.Write(body.Bytes())
	_, err = packet.WriteTo(w)
	return err
}

func (p *SubackPacket) Unpack(b io.Reader) (err error) {
	p.MessageID, err = decodeUint16(b)
	if err != nil {
		return err
	}
	payloadLength := p.FixedHeader.RemainingLength - 2

	for payloadLength > 0 {
		p.ReturnCodes = make([]byte, payloadLength)
		_, err := io.ReadFull(b, p.ReturnCodes)
		return err
	}
	return nil
}
