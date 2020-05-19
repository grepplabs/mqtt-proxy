package mqttcodec

import (
	"bytes"
	"fmt"
	"io"
)

type UnsubscribePacket struct {
	FixedHeader
	MessageID    uint16
	TopicFilters []string
}

func (p *UnsubscribePacket) Type() byte {
	return p.MessageType
}

func (p *UnsubscribePacket) Name() string {
	return "UNSUBSCRIBE"
}

func (p *UnsubscribePacket) String() string {
	return fmt.Sprintf("%s MessageID: %d %+v", p.FixedHeader, p.MessageID, p.TopicFilters)
}

func (p *UnsubscribePacket) Write(w io.Writer) (err error) {
	var body bytes.Buffer

	body.Write(encodeUint16(p.MessageID))
	for _, topicFilter := range p.TopicFilters {
		body.Write(encodeString(topicFilter))
	}
	p.FixedHeader.RemainingLength = body.Len()
	packet := p.FixedHeader.pack()
	packet.Write(body.Bytes())
	_, err = packet.WriteTo(w)
	return err
}

func (p *UnsubscribePacket) Unpack(b io.Reader) (err error) {
	p.MessageID, err = decodeUint16(b)
	if err != nil {
		return err
	}
	payloadLength := p.FixedHeader.RemainingLength - 2
	for payloadLength > 0 {
		topicFilter, err := decodeString(b)
		if err != nil {
			return err
		}
		p.TopicFilters = append(p.TopicFilters, topicFilter)
		payloadLength -= 2 + len(topicFilter)
	}
	return nil
}
