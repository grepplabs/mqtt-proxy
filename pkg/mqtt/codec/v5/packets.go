package v5

import (
	"bytes"
	"fmt"
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

func ReadPacket(r io.Reader) (mqttproto.ControlPacket, error) {
	var fh mqttproto.FixedHeader
	b1 := make([]byte, 1)
	_, err := io.ReadFull(r, b1)
	if err != nil {
		return nil, err
	}

	err = fh.Unpack(b1[0], r)
	if err != nil {
		return nil, err
	}

	err = fh.Validate()
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

func NewControlPacket(packetType byte) mqttproto.ControlPacket {
	switch packetType {
	case mqttproto.CONNECT:
		return &ConnectPacket{FixedHeader: mqttproto.FixedHeader{MessageType: mqttproto.CONNECT}}
	case mqttproto.CONNACK:
		return &ConnackPacket{FixedHeader: mqttproto.FixedHeader{MessageType: mqttproto.CONNACK}}
	case mqttproto.PUBLISH:
		return &PublishPacket{FixedHeader: mqttproto.FixedHeader{MessageType: mqttproto.PUBLISH}}
	case mqttproto.PUBACK:
		return &PubackPacket{FixedHeader: mqttproto.FixedHeader{MessageType: mqttproto.PUBACK}}
	case mqttproto.PUBREC:
		return &PubrecPacket{FixedHeader: mqttproto.FixedHeader{MessageType: mqttproto.PUBREC}}
	case mqttproto.PUBREL:
		return &PubrelPacket{FixedHeader: mqttproto.FixedHeader{MessageType: mqttproto.PUBREL}}
	case mqttproto.PUBCOMP:
		return &PubcompPacket{FixedHeader: mqttproto.FixedHeader{MessageType: mqttproto.PUBCOMP}}
	case mqttproto.PINGREQ:
		return &PingreqPacket{FixedHeader: mqttproto.FixedHeader{MessageType: mqttproto.PINGREQ}}
	case mqttproto.PINGRESP:
		return &PingrespPacket{FixedHeader: mqttproto.FixedHeader{MessageType: mqttproto.PINGRESP}}
	case mqttproto.DISCONNECT:
		return &DisconnectPacket{FixedHeader: mqttproto.FixedHeader{MessageType: mqttproto.DISCONNECT}}
	}
	return nil
}

func NewControlPacketWithHeader(fh mqttproto.FixedHeader) (mqttproto.ControlPacket, error) {
	switch fh.MessageType {
	case mqttproto.CONNECT:
		return &ConnectPacket{FixedHeader: fh}, nil
	case mqttproto.CONNACK:
		return &ConnackPacket{FixedHeader: fh}, nil
	case mqttproto.PUBLISH:
		return &PublishPacket{FixedHeader: fh}, nil
	case mqttproto.PUBACK:
		return &PubackPacket{FixedHeader: fh}, nil
	case mqttproto.PUBREC:
		return &PubrecPacket{FixedHeader: fh}, nil
	case mqttproto.PUBREL:
		return &PubrelPacket{FixedHeader: fh}, nil
	case mqttproto.PUBCOMP:
		return &PubcompPacket{FixedHeader: fh}, nil
	case mqttproto.PINGREQ:
		return &PingreqPacket{FixedHeader: fh}, nil
	case mqttproto.PINGRESP:
		return &PingrespPacket{FixedHeader: fh}, nil
	case mqttproto.DISCONNECT:
		return &DisconnectPacket{FixedHeader: fh}, nil
	default:
		return nil, fmt.Errorf("unsupported packet type 0x%x", fh.MessageType)
	}
}
