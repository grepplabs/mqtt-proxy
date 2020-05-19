package mqttcodec

import (
	"encoding/binary"
	"io"
)

func decodeByte(r io.Reader) (byte, error) {
	b := make([]byte, 1)
	_, err := r.Read(b)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func decodeString(r io.Reader) (string, error) {
	b, err := decodeBytes(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func encodeString(s string) []byte {
	return encodeBytes([]byte(s))
}

func decodeBytes(r io.Reader) ([]byte, error) {
	length, err := decodeUint16(r)
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

func encodeBytes(b []byte) []byte {
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(b)))
	return append(length, b...)
}

func decodeUint16(r io.Reader) (uint16, error) {
	b := make([]byte, 2)
	_, err := io.ReadFull(r, b)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b), nil
}

func encodeUint16(v uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	return b
}
