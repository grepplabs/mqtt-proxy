package proto

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixedHeaderCodec(t *testing.T) {
	tests := []struct {
		name       string
		header     FixedHeader
		encodedHex string
	}{
		{
			name:       "remaining length 0 (1 byte)",
			header:     FixedHeader{},
			encodedHex: "0000",
		},
		{
			name:       "remaining length 1 (1 byte)",
			header:     FixedHeader{RemainingLength: 1},
			encodedHex: "0001",
		},
		{
			name:       "remaining length 63 (1 byte)",
			header:     FixedHeader{RemainingLength: 63},
			encodedHex: "003f",
		},
		{
			name:       "remaining length 127 (1 byte)",
			header:     FixedHeader{RemainingLength: 127},
			encodedHex: "007f",
		},
		{
			name:       "remaining length 128 (2 bytes)",
			header:     FixedHeader{RemainingLength: 128},
			encodedHex: "008001",
		},
		{
			name:       "remaining length 129 (2 bytes)",
			header:     FixedHeader{RemainingLength: 129},
			encodedHex: "008101",
		},
		{
			name:       "remaining length 16383 (2 bytes)",
			header:     FixedHeader{RemainingLength: 16383},
			encodedHex: "00ff7f",
		},
		{
			name:       "remaining length 16384 (3 bytes)",
			header:     FixedHeader{RemainingLength: 16384},
			encodedHex: "00808001",
		},
		{
			name:       "remaining length 2097151 (3 bytes)",
			header:     FixedHeader{RemainingLength: 2097151},
			encodedHex: "00ffff7f",
		},
		{
			name:       "remaining length 2097152 (4 bytes)",
			header:     FixedHeader{RemainingLength: 2097152},
			encodedHex: "0080808001",
		},
		{
			name:       "remaining length 268435455 (4 bytes)",
			header:     FixedHeader{RemainingLength: maxRemainingLength},
			encodedHex: "00ffffff7f",
		},
		{
			name:       "Connect Message",
			header:     FixedHeader{MessageType: CONNECT, RemainingLength: 18},
			encodedHex: "1012",
		},
		{
			name:       "Publish Message",
			header:     FixedHeader{MessageType: PUBLISH},
			encodedHex: "3000",
		},
		{
			name:       "Publish Message Retain",
			header:     FixedHeader{MessageType: PUBLISH, Retain: true},
			encodedHex: "3100",
		},
		{
			name:       "Publish Message QoS 1",
			header:     FixedHeader{MessageType: PUBLISH, Qos: AT_LEAST_ONCE},
			encodedHex: "3200",
		},
		{
			name:       "Publish Message QoS 1 Retain",
			header:     FixedHeader{MessageType: PUBLISH, Qos: AT_LEAST_ONCE, Retain: true},
			encodedHex: "3300",
		},
		{
			name:       "Publish Message QoS 1 duplicate",
			header:     FixedHeader{MessageType: PUBLISH, Qos: AT_LEAST_ONCE, Dup: true},
			encodedHex: "3a00",
		},
		{
			name:       "Publish Message QoS 2",
			header:     FixedHeader{MessageType: PUBLISH, Qos: EXACTLY_ONCE},
			encodedHex: "3400",
		},
		{
			name:       "Publish Message QoS 2 Retain",
			header:     FixedHeader{MessageType: PUBLISH, Qos: EXACTLY_ONCE, Retain: true},
			encodedHex: "3500",
		},
		{
			name:       "Publish Message QoS 2 duplicate",
			header:     FixedHeader{MessageType: PUBLISH, Qos: EXACTLY_ONCE, Dup: true},
			encodedHex: "3c00",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			t.Log(tc.header)

			buffer := tc.header.Pack()
			encodedBytes := buffer.Bytes()
			a.Equal(tc.encodedHex, hex.EncodeToString(encodedBytes))

			header := FixedHeader{}
			err := header.decode(bytes.NewBuffer(encodedBytes))
			if err != nil {
				t.Fatal(err)
			}

			a.Equal(tc.header, header)
			a.Nil(header.Validate())
		})
	}
}

func TestFixedHeaderDecodeError(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remaining length too long",
			input:    "008080808001",
			expected: "the maximum number of bytes in the length is 4, but was 5",
		},
		{
			name:     "remaining length too long",
			input:    "00808080808001",
			expected: "the maximum number of bytes in the length is 4, but was 6",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			input, err := hex.DecodeString(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			header := FixedHeader{}
			err = header.decode(bytes.NewBuffer(input))
			a := assert.New(t)
			a.EqualError(err, tc.expected)
		})
	}
}

func TestFixedHeaderValidationError(t *testing.T) {
	tests := []struct {
		name     string
		input    FixedHeader
		expected string
	}{
		{
			name:     "remaining length too long",
			input:    FixedHeader{RemainingLength: maxRemainingLength + 1},
			expected: "the maximum remaining length is 268435455, but was 268435456",
		},
		{
			name:     "negative remaining length",
			input:    FixedHeader{RemainingLength: -1},
			expected: "negative remaining length -1",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()
			a := assert.New(t)
			a.EqualError(err, tc.expected)
		})
	}
}
