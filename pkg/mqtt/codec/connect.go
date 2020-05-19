package mqttcodec

import (
	"bytes"
	"fmt"
	"io"
)

type ConnectPacket struct {
	FixedHeader

	ProtocolName     string
	ProtocolLevel    byte
	HasUsername      bool
	HasPassword      bool
	WillRetain       bool
	WillQos          byte
	WillFlag         bool
	CleanSession     bool
	ReservedBit      bool
	KeepAliveSeconds uint16

	ClientIdentifier string
	WillTopic        string
	WillMessage      []byte
	Username         string
	Password         []byte
}

func (p *ConnectPacket) Type() byte {
	return p.MessageType
}
func (p *ConnectPacket) Name() string {
	return "CONNECT"
}

func (p *ConnectPacket) String() string {
	// Password is not provided
	return fmt.Sprintf("%s ProtocolName: %s ProtocolLevel: %d  CleanSession: %t WillFlag: %t WillQos: %d WillRetain: %t HasUsername: %t HasPassword: %t KeepAliveSeconds: %d ClientID: %s WillTopic: %s Willmessage: %s Username: %s", p.FixedHeader, p.ProtocolName, p.ProtocolLevel, p.CleanSession, p.WillFlag, p.WillQos, p.WillRetain, p.HasUsername, p.HasPassword, p.KeepAliveSeconds, p.ClientIdentifier, p.WillTopic, p.WillMessage, p.Username)
}

func (p *ConnectPacket) Write(w io.Writer) (err error) {
	var body bytes.Buffer

	body.Write(encodeString(p.ProtocolName))
	body.WriteByte(p.ProtocolLevel)

	body.WriteByte(p.getConnectFlags())
	body.Write(encodeUint16(p.KeepAliveSeconds))
	body.Write(encodeString(p.ClientIdentifier))
	if p.WillFlag {
		body.Write(encodeString(p.WillTopic))
		body.Write(encodeBytes(p.WillMessage))
	}
	if p.HasUsername {
		body.Write(encodeString(p.Username))
	}
	if p.HasPassword {
		body.Write(encodeBytes(p.Password))
	}

	p.FixedHeader.RemainingLength = body.Len()
	packet := p.FixedHeader.pack()
	packet.Write(body.Bytes())
	_, err = packet.WriteTo(w)
	return err
}

func (p *ConnectPacket) Unpack(r io.Reader) (err error) {
	// variable header
	p.ProtocolName, err = decodeString(r)
	if err != nil {
		return err
	}
	version, err := decodeByte(r)
	if err != nil {
		return err
	}
	p.ProtocolLevel = version
	connectFlags, err := decodeByte(r)
	if err != nil {
		return err
	}
	p.HasUsername = (connectFlags & 0x80) == 0x80
	p.HasPassword = (connectFlags & 0x40) == 0x40
	p.WillRetain = (connectFlags & 0x20) == 0x20
	p.WillQos = (connectFlags & 0x18) >> 3
	p.WillFlag = (connectFlags & 0x04) == 0x04
	p.CleanSession = (connectFlags & 0x02) == 0x02
	p.ReservedBit = (connectFlags & 0x01) == 0x01

	p.KeepAliveSeconds, err = decodeUint16(r)
	if err != nil {
		return err
	}
	// payload
	p.ClientIdentifier, err = decodeString(r)
	if err != nil {
		return err
	}
	if p.WillFlag {
		p.WillTopic, err = decodeString(r)
		if err != nil {
			return err
		}
		p.WillMessage, err = decodeBytes(r)
		if err != nil {
			return err
		}
	}
	if p.HasUsername {
		p.Username, err = decodeString(r)
		if err != nil {
			return err
		}
	}
	if p.HasPassword {
		p.Password, err = decodeBytes(r)
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
	if p.CleanSession {
		connectFlags |= 0x02
	}
	return connectFlags
}
