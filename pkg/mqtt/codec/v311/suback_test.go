package v311

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

func TestNewSubackPacket(t *testing.T) {
	a := assert.New(t)
	packet := NewControlPacket(mqttproto.SUBACK).(*SubackPacket)
	a.Equal(mqttproto.SUBACK, packet.MessageType)
	a.Equal(mqttproto.MqttMessageTypeNames[packet.MessageType], packet.Name())
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
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.SUBACK,
					RemainingLength: 3,
				},
				MessageID:   1,
				ReturnCodes: []byte{mqttproto.AT_LEAST_ONCE},
			},
		},
		{
			name:       "2 granted qos 1",
			encodedHex: "900400010101",
			packet: &SubackPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.SUBACK,
					RemainingLength: 4,
				},
				MessageID:   1,
				ReturnCodes: []byte{mqttproto.AT_LEAST_ONCE, mqttproto.AT_LEAST_ONCE},
			},
		},
		{
			name:       "2 granted qos 2",
			encodedHex: "900400010202",
			packet: &SubackPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.SUBACK,
					RemainingLength: 4,
				},
				MessageID:   1,
				ReturnCodes: []byte{mqttproto.EXACTLY_ONCE, mqttproto.EXACTLY_ONCE},
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
