package v311

import (
	"bytes"
	"fmt"
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

type PublishPacket struct {
	mqttproto.FixedHeader
	TopicName string
	MessageID uint16
	Message   []byte
}

func (p *PublishPacket) Type() byte {
	return p.MessageType
}

func (p *PublishPacket) Version() byte {
	return mqttproto.MQTT_3_1_1
}

func (c *PublishPacket) Name() string {
	return "PUBLISH"
}

func (p *PublishPacket) String() string {
	return fmt.Sprintf("%v TopicName: %s MessageID: %d Message: %s", p.FixedHeader, p.TopicName, p.MessageID, string(p.Message))
}

func (p *PublishPacket) Write(w io.Writer) (err error) {
	var body bytes.Buffer

	body.Write(mqttproto.EncodeString(p.TopicName))
	if p.Qos > 0 {
		body.Write(mqttproto.EncodeUint16(p.MessageID))
	}

	p.FixedHeader.RemainingLength = body.Len() + len(p.Message)
	packet := p.FixedHeader.Pack()
	packet.Write(body.Bytes())
	packet.Write(p.Message)
	_, err = w.Write(packet.Bytes())
	return err
}

func (p *PublishPacket) Unpack(b io.Reader) (err error) {
	var payloadLength = p.FixedHeader.RemainingLength

	p.TopicName, err = mqttproto.DecodeString(b)
	if err != nil {
		return err
	}

	if p.Qos > 0 {
		p.MessageID, err = mqttproto.DecodeUint16(b)
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
