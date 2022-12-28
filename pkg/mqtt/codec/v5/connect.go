package v5

import (
	"bytes"
	"fmt"
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

type ConnectPacket struct {
	mqttproto.FixedHeader

	ProtocolName     string
	ProtocolLevel    byte
	HasUsername      bool
	HasPassword      bool
	WillRetain       bool
	WillQos          byte
	WillFlag         bool
	CleanStart       bool
	ReservedBit      bool
	KeepAliveSeconds uint16

	ConnectProperties Properties

	ClientIdentifier string
	WillProperties   Properties
	WillTopic        string
	WillPayload      []byte
	Username         string
	Password         []byte
}

func (p *ConnectPacket) Type() byte {
	return p.MessageType
}

func (p *ConnectPacket) Version() byte {
	return mqttproto.MQTT_5
}

func (p *ConnectPacket) Name() string {
	return "CONNECT"
}

func (p *ConnectPacket) String() string {
	// Password is not provided
	return fmt.Sprintf("%v ProtocolName: %s ProtocolLevel: %d  CleanStart: %t WillFlag: %t WillQos: %d WillRetain: %t HasUsername: %t HasPassword: %t KeepAliveSeconds: %d ClientID: %s WillTopic: %s WillPayload: %s Username: %s", p.FixedHeader, p.ProtocolName, p.ProtocolLevel, p.CleanStart, p.WillFlag, p.WillQos, p.WillRetain, p.HasUsername, p.HasPassword, p.KeepAliveSeconds, p.ClientIdentifier, p.WillTopic, p.WillPayload, p.Username)
}

func (p *ConnectPacket) Write(w io.Writer) (err error) {
	var body bytes.Buffer

	body.Write(mqttproto.EncodeString(p.ProtocolName))
	body.WriteByte(p.ProtocolLevel)

	body.WriteByte(p.getConnectFlags())
	body.Write(mqttproto.EncodeUint16(p.KeepAliveSeconds))
	// properties
	body.Write(p.ConnectProperties.Encode())
	// payload
	body.Write(mqttproto.EncodeString(p.ClientIdentifier))
	if p.WillFlag {
		body.Write(p.WillProperties.Encode())
		body.Write(mqttproto.EncodeString(p.WillTopic))
		body.Write(mqttproto.EncodeBytes(p.WillPayload))
	}
	if p.HasUsername {
		body.Write(mqttproto.EncodeString(p.Username))
	}
	if p.HasPassword {
		body.Write(mqttproto.EncodeBytes(p.Password))
	}

	p.FixedHeader.RemainingLength = body.Len()
	packet := p.FixedHeader.Pack()
	packet.Write(body.Bytes())
	_, err = packet.WriteTo(w)
	return err
}

func (p *ConnectPacket) Unpack(r io.Reader) (err error) {
	// variable header
	p.ProtocolName, err = mqttproto.DecodeString(r)
	if err != nil {
		return err
	}
	version, err := mqttproto.DecodeByte(r)
	if err != nil {
		return err
	}
	p.ProtocolLevel = version
	connectFlags, err := mqttproto.DecodeByte(r)
	if err != nil {
		return err
	}
	p.HasUsername = (connectFlags & 0x80) == 0x80
	p.HasPassword = (connectFlags & 0x40) == 0x40
	p.WillRetain = (connectFlags & 0x20) == 0x20
	p.WillQos = (connectFlags & 0x18) >> 3
	p.WillFlag = (connectFlags & 0x04) == 0x04
	p.CleanStart = (connectFlags & 0x02) == 0x02
	p.ReservedBit = (connectFlags & 0x01) == 0x01

	p.KeepAliveSeconds, err = mqttproto.DecodeUint16(r)
	if err != nil {
		return err
	}
	// properties
	err = p.ConnectProperties.Unpack(r)
	if err != nil {
		return err
	}
	// payload
	p.ClientIdentifier, err = mqttproto.DecodeString(r)
	if err != nil {
		return err
	}
	if p.WillFlag {
		err = p.WillProperties.Unpack(r)
		if err != nil {
			return err
		}
		p.WillTopic, err = mqttproto.DecodeString(r)
		if err != nil {
			return err
		}
		p.WillPayload, err = mqttproto.DecodeBytes(r)
		if err != nil {
			return err
		}
	}
	if p.HasUsername {
		p.Username, err = mqttproto.DecodeString(r)
		if err != nil {
			return err
		}
	}
	if p.HasPassword {
		p.Password, err = mqttproto.DecodeBytes(r)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *ConnectPacket) getConnectFlags() byte {
	var connectFlags byte
	if p.HasUsername {
		connectFlags |= 0x80
	}
	if p.HasPassword {
		connectFlags |= 0x40
	}
	if p.WillRetain {
		connectFlags |= 0x20
	}
	connectFlags |= (p.WillQos & 0x03) << 3
	if p.WillFlag {
		connectFlags |= 0x04
	}
	if p.CleanStart {
		connectFlags |= 0x02
	}
	return connectFlags
}
