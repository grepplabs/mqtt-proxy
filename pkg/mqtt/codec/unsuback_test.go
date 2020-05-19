package mqttcodec

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUnsubackPacket(t *testing.T) {
	a := assert.New(t)
	packet := NewControlPacket(UNSUBACK).(*UnsubackPacket)
	a.Equal(UNSUBACK, packet.MessageType)
	a.Equal(MqttMessageTypeNames[packet.MessageType], packet.Name())
	t.Log(packet)
}

func TestUnsubackPacketCodec(t *testing.T) {
	tests := []struct {
		name       string
		encodedHex string
		packet     *UnsubackPacket
	}{
		{
			name:       "message 2",
			encodedHex: "b0020002",
			packet: &UnsubackPacket{
				FixedHeader: FixedHeader{
					MessageType:     UNSUBACK,
					RemainingLength: 2,
				},
				MessageID: 2,
			},
		},
		{
			name:       "message 1024",
			encodedHex: "b0020400",
			packet: &UnsubackPacket{
				FixedHeader: FixedHeader{
					MessageType:     UNSUBACK,
					RemainingLength: 2,
				},
				MessageID: 1024,
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
			packet := decoded.(*UnsubackPacket)
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
