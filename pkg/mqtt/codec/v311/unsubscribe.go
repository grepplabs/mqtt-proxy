package v311

import (
	"bytes"
	"fmt"
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

type UnsubscribePacket struct {
	mqttproto.FixedHeader
	MessageID    uint16
	TopicFilters []string
}

func (p *UnsubscribePacket) Type() byte {
	return p.MessageType
}

func (p *UnsubscribePacket) Version() byte {
	return mqttproto.MQTT_3_1_1
}

func (p *UnsubscribePacket) Name() string {
	return "UNSUBSCRIBE"
}

func (p *UnsubscribePacket) String() string {
	return fmt.Sprintf("%v MessageID: %d %+v", p.FixedHeader, p.MessageID, p.TopicFilters)
}

func (p *UnsubscribePacket) Write(w io.Writer) (err error) {
	var body bytes.Buffer

	body.Write(mqttproto.EncodeUint16(p.MessageID))
	for _, topicFilter := range p.TopicFilters {
		body.Write(mqttproto.EncodeString(topicFilter))
	}
	p.FixedHeader.RemainingLength = body.Len()
	packet := p.FixedHeader.Pack()
	packet.Write(body.Bytes())
	_, err = packet.WriteTo(w)
	return err
}

func (p *UnsubscribePacket) Unpack(b io.Reader) (err error) {
	p.MessageID, err = mqttproto.DecodeUint16(b)
	if err != nil {
		return err
	}
	payloadLength := p.FixedHeader.RemainingLength - 2
	for payloadLength > 0 {
		topicFilter, err := mqttproto.DecodeString(b)
		if err != nil {
			return err
		}
		p.TopicFilters = append(p.TopicFilters, topicFilter)
		payloadLength -= 2 + len(topicFilter)
	}
	return nil
}
