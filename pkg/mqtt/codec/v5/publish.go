package v5

import (
	"bytes"
	"fmt"
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

type PublishPacket struct {
	mqttproto.FixedHeader
	TopicName         string
	MessageID         uint16
	PublishProperties Properties
	Message           []byte
}

func (p *PublishPacket) Type() byte {
	return p.MessageType
}

func (p *PublishPacket) Version() byte {
	return mqttproto.MQTT_5
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
	body.Write(p.PublishProperties.Encode())

	p.FixedHeader.RemainingLength = body.Len() + len(p.Message)
	packet := p.FixedHeader.Pack()
	packet.Write(body.Bytes())
	packet.Write(p.Message)
	_, err = w.Write(packet.Bytes())
	return err
}

func (p *PublishPacket) Unpack(r io.Reader) (err error) {
	var payloadLength = p.FixedHeader.RemainingLength

	cr := &mqttproto.CountingReader{Reader: r}
	p.TopicName, err = mqttproto.DecodeString(cr)
	if err != nil {
		return err
	}
	if p.Qos > 0 {
		p.MessageID, err = mqttproto.DecodeUint16(cr)
		if err != nil {
			return err
		}
	}
	err = p.PublishProperties.Unpack(cr)
	if err != nil {
		return err
	}
	payloadLength -= cr.BytesRead
	if payloadLength < 0 {
		return fmt.Errorf("error unpacking publish, payload length < 0")
	}
	p.Message = make([]byte, payloadLength)
	_, err = io.ReadFull(cr, p.Message)
	return err
}
