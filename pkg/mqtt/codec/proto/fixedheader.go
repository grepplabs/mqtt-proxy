package proto

import (
	"bytes"
	"fmt"
	"io"
)

const maxRemainingLength = 268435455

type FixedHeader struct {
	MessageType     byte
	Dup             bool
	Qos             byte
	Retain          bool
	RemainingLength int
}

func (fh *FixedHeader) MessageName() string {
	return MqttMessageTypeNames[fh.MessageType]
}

func (fh *FixedHeader) Validate() error {
	if fh.RemainingLength < 0 {
		return fmt.Errorf("negative remaining length %d", fh.RemainingLength)
	}
	if fh.RemainingLength > maxRemainingLength {
		return fmt.Errorf("the maximum remaining length is %d, but was %d", maxRemainingLength, fh.RemainingLength)
	}
	return nil
}

func (fh *FixedHeader) decode(r io.Reader) (err error) {
	b1 := make([]byte, 1)
	_, err = io.ReadFull(r, b1)
	if err != nil {
		return err
	}
	return fh.Unpack(b1[0], r)
}

func (fh *FixedHeader) Unpack(typeAndFlags byte, r io.Reader) (err error) {
	fh.MessageType = typeAndFlags >> 4
	fh.Dup = (typeAndFlags & 0x08) == 0x08
	fh.Qos = (typeAndFlags & 0x06) >> 1
	fh.Retain = (typeAndFlags & 0x01) != 0
	fh.RemainingLength, err = DecodeUvarint(r)
	return err
}

func (fh *FixedHeader) Pack() bytes.Buffer {
	var header bytes.Buffer
	header.WriteByte(fh.getFixedHeaderByte1())
	WriteUvarint(&header, uint32(fh.RemainingLength))
	return header
}

func (fh *FixedHeader) String() string {
	return fmt.Sprintf("%s: Dup: %t QoS: %s Retain: %t rLength: %d", MqttMessageTypeNames[fh.MessageType], fh.Dup, MqttQoSNames[fh.Qos], fh.Retain, fh.RemainingLength)
}

func (fh *FixedHeader) getFixedHeaderByte1() byte {
	var result byte
	result |= fh.MessageType << 4
	if fh.Dup {
		result |= 0x08
	}
	result |= fh.Qos << 1
	if fh.Retain {
		result |= 0x01
	}
	return result
}
