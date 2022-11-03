package mqttcodec

import (
	"bytes"
	"fmt"
	"io"
)

type ControlPacket interface {
	Write(io.Writer) error
	Unpack(io.Reader) error
	String() string
	Type() byte
	Name() string
}

func ReadPacket(r io.Reader) (ControlPacket, error) {
	var fh FixedHeader
	b1 := make([]byte, 1)
	_, err := io.ReadFull(r, b1)
	if err != nil {
		return nil, err
	}

	err = fh.unpack(b1[0], r)
	if err != nil {
		return nil, err
	}

	err = fh.validate()
	if err != nil {
		return nil, err
	}

	cp, err := NewControlPacketWithHeader(fh)
	if err != nil {
		return nil, err
	}

	packetBytes := make([]byte, fh.RemainingLength)
	n, err := io.ReadFull(r, packetBytes)
	if err != nil {
		return nil, err
	}
	if n != fh.RemainingLength {
		return nil, fmt.Errorf("failed to read encoded data, read %d from %d", n, fh.RemainingLength)
	}
	err = cp.Unpack(bytes.NewBuffer(packetBytes))
	return cp, err
}

func NewControlPacket(packetType byte) ControlPacket {
	switch packetType {
	case CONNECT:
		return &ConnectPacket{FixedHeader: FixedHeader{MessageType: CONNECT}}
	case CONNACK:
		return &ConnackPacket{FixedHeader: FixedHeader{MessageType: CONNACK}}
	case PUBLISH:
		return &PublishPacket{FixedHeader: FixedHeader{MessageType: PUBLISH}}
	case PUBACK:
		return &PubackPacket{FixedHeader: FixedHeader{MessageType: PUBACK}}
	case PUBREC:
		return &PubrecPacket{FixedHeader: FixedHeader{MessageType: PUBREC}}
	case PUBREL:
		return &PubrelPacket{FixedHeader: FixedHeader{MessageType: PUBREL}}
	case PUBCOMP:
		return &PubcompPacket{FixedHeader: FixedHeader{MessageType: PUBCOMP}}
	case SUBSCRIBE:
		return &SubscribePacket{FixedHeader: FixedHeader{MessageType: SUBSCRIBE}}
	case SUBACK:
		return &SubackPacket{FixedHeader: FixedHeader{MessageType: SUBACK}}
	case UNSUBSCRIBE:
		return &UnsubscribePacket{FixedHeader: FixedHeader{MessageType: UNSUBSCRIBE}}
	case UNSUBACK:
		return &UnsubackPacket{FixedHeader: FixedHeader{MessageType: UNSUBACK}}
	case PINGREQ:
		return &PingreqPacket{FixedHeader: FixedHeader{MessageType: PINGREQ}}
	case PINGRESP:
		return &PingrespPacket{FixedHeader: FixedHeader{MessageType: PINGRESP}}
	case DISCONNECT:
		return &DisconnectPacket{FixedHeader: FixedHeader{MessageType: DISCONNECT}}

	}
	return nil
}

func NewControlPacketWithHeader(fh FixedHeader) (ControlPacket, error) {
	switch fh.MessageType {
	case CONNECT:
		return &ConnectPacket{FixedHeader: fh}, nil
	case CONNACK:
		return &ConnackPacket{FixedHeader: fh}, nil
	case PUBLISH:
		return &PublishPacket{FixedHeader: fh}, nil
	case PUBACK:
		return &PubackPacket{FixedHeader: fh}, nil
	case PUBREC:
		return &PubrecPacket{FixedHeader: fh}, nil
	case PUBREL:
		return &PubrelPacket{FixedHeader: fh}, nil
	case PUBCOMP:
		return &PubcompPacket{FixedHeader: fh}, nil
	case SUBSCRIBE:
		return &SubscribePacket{FixedHeader: fh}, nil
	case SUBACK:
		return &SubackPacket{FixedHeader: fh}, nil
	case UNSUBSCRIBE:
		return &UnsubscribePacket{FixedHeader: fh}, nil
	case UNSUBACK:
		return &UnsubackPacket{FixedHeader: fh}, nil
	case PINGREQ:
		return &PingreqPacket{FixedHeader: fh}, nil
	case PINGRESP:
		return &PingrespPacket{FixedHeader: fh}, nil
	case DISCONNECT:
		return &DisconnectPacket{FixedHeader: fh}, nil
	default:
		return nil, fmt.Errorf("unsupported packet type 0x%x", fh.MessageType)
	}
}
