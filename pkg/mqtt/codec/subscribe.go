package mqttcodec

import (
	"bytes"
	"fmt"
	"io"
)

type TopicSubscription struct {
	TopicFilter string
	Qos         byte
}

func (ts *TopicSubscription) String() string {
	return fmt.Sprintf("TopicFilter: %s Qos: %d", ts.TopicFilter, ts.Qos)
}

type SubscribePacket struct {
	FixedHeader
	MessageID          uint16
	TopicSubscriptions []TopicSubscription
}

func (p *SubscribePacket) Type() byte {
	return p.MessageType
}

func (p *SubscribePacket) Name() string {
	return "SUBSCRIBE"
}

func (p *SubscribePacket) String() string {
	return fmt.Sprintf("%s MessageID: %d %+v", p.FixedHeader, p.MessageID, p.TopicSubscriptions)
}

func (p *SubscribePacket) Write(w io.Writer) (err error) {
	var body bytes.Buffer

	body.Write(encodeUint16(p.MessageID))
	for _, ts := range p.TopicSubscriptions {
		body.Write(encodeString(ts.TopicFilter))
		body.WriteByte(ts.Qos)
	}
	p.FixedHeader.RemainingLength = body.Len()
	packet := p.FixedHeader.pack()
	packet.Write(body.Bytes())
	_, err = packet.WriteTo(w)
	return err
}

func (p *SubscribePacket) Unpack(b io.Reader) (err error) {
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
		qos, err := decodeByte(b)
		if err != nil {
			return err
		}
		p.TopicSubscriptions = append(p.TopicSubscriptions, TopicSubscription{
			TopicFilter: topicFilter,
			Qos:         qos,
		})
		payloadLength -= 3 + len(topicFilter)
	}
	return nil
}
