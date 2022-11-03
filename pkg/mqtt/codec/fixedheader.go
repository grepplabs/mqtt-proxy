package mqttcodec

import (
	"bytes"
	"encoding/binary"
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

func (fh *FixedHeader) validate() error {
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
	return fh.unpack(b1[0], r)
}

func (fh *FixedHeader) unpack(typeAndFlags byte, r io.Reader) (err error) {
	fh.MessageType = typeAndFlags >> 4
	fh.Dup = (typeAndFlags & 0x08) == 0x08
	fh.Qos = (typeAndFlags & 0x06) >> 1
	fh.Retain = (typeAndFlags & 0x01) != 0
	fh.RemainingLength, err = decodeLength(r)
	return err
}

func (fh *FixedHeader) pack() bytes.Buffer {
	var header bytes.Buffer
	header.WriteByte(fh.getFixedHeaderByte1())
	writeLength(&header, uint32(fh.RemainingLength))
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

// modified binary.PutUvarint
func writeLength(buffer *bytes.Buffer, x uint32) int {
	i := 0
	for x >= 0x80 {
		buffer.WriteByte(byte(x) | 0x80)
		x >>= 7
		i++
	}
	buffer.WriteByte(byte(x))
	return i + 1
}

func decodeLength(r io.Reader) (int, error) {
	byteReader := newByteReader(r)
	remainingLength, err := binary.ReadUvarint(byteReader)
	if err != nil {
		return 0, err
	}
	if byteReader.bytesRead > 4 {
		return 0, fmt.Errorf("the maximum number of bytes in the remaining length is 4, but was %d", byteReader.bytesRead)
	}
	return int(remainingLength), nil
}

func newByteReader(r io.Reader) *byteReader {
	return &byteReader{reader: r}
}

type byteReader struct {
	reader    io.Reader
	bytesRead int
	buf       [1]byte
}

func (r *byteReader) ReadByte() (byte, error) {
	n, err := io.ReadFull(r.reader, r.buf[:])
	if err != nil {
		return 0, err
	}
	r.bytesRead += n
	return r.buf[0], nil
}
