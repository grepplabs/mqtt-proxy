package mqttcodec

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSubackPacket(t *testing.T) {
	a := assert.New(t)
	packet := NewControlPacket(SUBACK).(*SubackPacket)
	a.Equal(SUBACK, packet.MessageType)
	a.Equal(MqttMessageTypeNames[packet.MessageType], packet.Name())
	t.Log(packet)
}

func TestSubackPacketCodec(t *testing.T) {
	tests := []struct {
		name       string
		encodedHex string
		packet     *SubackPacket
	}{
		{
			name:       "granted qos 1",
			encodedHex: "9003000101",
			packet: &SubackPacket{
				FixedHeader: FixedHeader{
					MessageType:     SUBACK,
					RemainingLength: 3,
				},
				MessageID:   1,
				ReturnCodes: []byte{AT_LEAST_ONCE},
			},
		},
		{
			name:       "2 granted qos 1",
			encodedHex: "900400010101",
			packet: &SubackPacket{
				FixedHeader: FixedHeader{
					MessageType:     SUBACK,
					RemainingLength: 4,
				},
				MessageID:   1,
				ReturnCodes: []byte{AT_LEAST_ONCE, AT_LEAST_ONCE},
			},
		},
		{
			name:       "2 granted qos 2",
			encodedHex: "900400010202",
			packet: &SubackPacket{
				FixedHeader: FixedHeader{
					MessageType:     SUBACK,
					RemainingLength: 4,
				},
				MessageID:   1,
				ReturnCodes: []byte{EXACTLY_ONCE, EXACTLY_ONCE},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			t.Log(tc.packet)

			// decode
			encodedBytes, err := hex.DecodeString(tc.encodedHex)
			if err != nil {
				t.Fatal(err)
			}
			r := bytes.NewReader(encodedBytes)
			decoded, err := ReadPacket(r)
			if err != nil {
				t.Fatal(err)
			}
			packet := decoded.(*SubackPacket)
			a.Equal(*tc.packet, *packet)

			// encode
			var output bytes.Buffer
			err = packet.Write(&output)
			if err != nil {
				t.Fatal(err)
			}
			encodedBytes = output.Bytes()
			a.Equal(tc.encodedHex, hex.EncodeToString(encodedBytes))
		})
	}
}
