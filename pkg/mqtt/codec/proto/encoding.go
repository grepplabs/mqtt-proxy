package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

func DecodeByte(r io.Reader) (byte, error) {
	b := make([]byte, 1)
	_, err := r.Read(b)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func DecodeString(r io.Reader) (string, error) {
	b, err := DecodeBytes(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func EncodeString(s string) []byte {
	return EncodeBytes([]byte(s))
}

func DecodeBytes(r io.Reader) ([]byte, error) {
	length, err := DecodeUint16(r)
	if err != nil {
		return nil, err
	}
	b := make([]byte, length)
	_, err = io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func EncodeBytes(b []byte) []byte {
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(b)))
	return append(length, b...)
}

func DecodeUint16(r io.Reader) (uint16, error) {
	b := make([]byte, 2)
	_, err := io.ReadFull(r, b)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b), nil
}

func EncodeUint16(v uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	return b
}

func DecodeUvarint(r io.Reader) (int, error) {
	byteReader := newByteReader(r)
	length, err := binary.ReadUvarint(byteReader)
	if err != nil {
		return 0, err
	}
	if byteReader.bytesRead > 4 {
		return 0, fmt.Errorf("the maximum number of bytes in the variable byte integer is 4, but was %d", byteReader.bytesRead)
	}
	return int(length), nil
}

// WriteUvarint is a modified binary.PutUvarint
func WriteUvarint(buffer *bytes.Buffer, x uint32) int {
	i := 0
	for x >= 0x80 {
		buffer.WriteByte(byte(x) | 0x80)
		x >>= 7
		i++
	}
	buffer.WriteByte(byte(x))
	return i + 1
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

type CountingReader struct {
	Reader    io.Reader
	BytesRead int
}

func (r *CountingReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.BytesRead += n
	return n, err
}
