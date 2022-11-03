package mqttcodec

import (
	"bytes"
	"fmt"
	"io"
)

type PublishPacket struct {
	FixedHeader
	TopicName string
	MessageID uint16
	Message   []byte
}

func (p *PublishPacket) Type() byte {
	return p.MessageType
}

func (c *PublishPacket) Name() string {
	return "PUBLISH"
}

func (p *PublishPacket) String() string {
	return fmt.Sprintf("%v TopicName: %s MessageID: %d Message: %s", p.FixedHeader, p.TopicName, p.MessageID, string(p.Message))
}

func (p *PublishPacket) Write(w io.Writer) (err error) {
	var body bytes.Buffer

	body.Write(encodeString(p.TopicName))
	if p.Qos > 0 {
		body.Write(encodeUint16(p.MessageID))
	}

	p.FixedHeader.RemainingLength = body.Len() + len(p.Message)
	packet := p.FixedHeader.pack()
	packet.Write(body.Bytes())
	packet.Write(p.Message)
	_, err = w.Write(packet.Bytes())
	return err
}

func (p *PublishPacket) Unpack(b io.Reader) (err error) {
	var payloadLength = p.FixedHeader.RemainingLength

	p.TopicName, err = decodeString(b)
	if err != nil {
		return err
	}

	if p.Qos > 0 {
		p.MessageID, err = decodeUint16(b)
		if err != nil {
			return err
		}
		payloadLength -= len(p.TopicName) + 4
	} else {
		payloadLength -= len(p.TopicName) + 2
	}
	if payloadLength < 0 {
		return fmt.Errorf("error unpacking publish, payload length < 0")
	}
	p.Message = make([]byte, payloadLength)
	_, err = io.ReadFull(b, p.Message)
	return err
}
