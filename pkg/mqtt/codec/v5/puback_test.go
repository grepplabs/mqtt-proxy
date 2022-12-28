package v5

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

func TestNewPubackPacket(t *testing.T) {
	a := assert.New(t)
	packet := NewControlPacket(mqttproto.PUBACK).(*PubackPacket)
	a.Equal(mqttproto.PUBACK, packet.MessageType)
	a.Equal(mqttproto.MqttMessageTypeNames[packet.MessageType], packet.Name())
	a.Equal(mqttproto.MQTT_5, packet.Version())
	t.Log(packet)
}

func TestPubackPacketCodec(t *testing.T) {
	tests := []struct {
		name       string
		encodedHex string
		packet     *PubackPacket
	}{
		{
			name:       "message 1024",
			encodedHex: "40020400",
			packet: &PubackPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBACK,
					RemainingLength: 2,
				},
				MessageID:        1024,
				ReasonCode:       0x00,
				PubackProperties: Properties{RawData: []byte{}},
			},
		},
		{
			name:       "response code 128, no properties",
			encodedHex: "400404008000",
			packet: &PubackPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBACK,
					RemainingLength: 4,
				},
				MessageID:        1024,
				ReasonCode:       0x80,
				PubackProperties: Properties{RawData: []byte{}},
			},
		},
		{
			name:       "response code 0, with properties",
			encodedHex: "400f0400800b2600036161610003626262",
			packet: &PubackPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBACK,
					RemainingLength: 15,
				},
				MessageID:        1024,
				ReasonCode:       0x80,
				PubackProperties: Properties{RawData: MustHexDecodeString("2600036161610003626262")},
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
			packet := decoded.(*PubackPacket)
			a.Equal(*tc.packet, *packet)
			a.Equal(mqttproto.MQTT_5, packet.Version())

			// encode
			var output bytes.Buffer
			err = packet.Write(&output)
			if err != nil {
				t.Fatal(err)
			}
			a.Equal(tc.packet.RemainingLength, packet.RemainingLength)
			encodedBytes = output.Bytes()
			a.Equal(tc.encodedHex, hex.EncodeToString(encodedBytes))
		})
	}
}
