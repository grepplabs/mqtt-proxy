package v311

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

func TestNewPubrelPacket(t *testing.T) {
	a := assert.New(t)
	packet := NewControlPacket(mqttproto.PUBREL).(*PubrelPacket)
	a.Equal(mqttproto.PUBREL, packet.MessageType)
	a.Equal(mqttproto.MqttMessageTypeNames[packet.MessageType], packet.Name())
	t.Log(packet)
}

func TestPubrelPacketCodec(t *testing.T) {
	tests := []struct {
		name       string
		encodedHex string
		packet     *PubrelPacket
	}{
		{
			name:       "message 1",
			encodedHex: "62020001",
			packet: &PubrelPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBREL,
					Qos:             mqttproto.AT_LEAST_ONCE,
					RemainingLength: 2,
				},
				MessageID: 1,
			},
		},
		{
			name:       "message 1024",
			encodedHex: "62020400",
			packet: &PubrelPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBREL,
					Qos:             mqttproto.AT_LEAST_ONCE,
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
			packet := decoded.(*PubrelPacket)
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
