package v5

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

func TestNewPubcompPacket(t *testing.T) {
	a := assert.New(t)
	packet := NewControlPacket(mqttproto.PUBCOMP).(*PubcompPacket)
	a.Equal(mqttproto.PUBCOMP, packet.MessageType)
	a.Equal(mqttproto.MqttMessageTypeNames[packet.MessageType], packet.Name())
	a.Equal(mqttproto.MQTT_5, packet.Version())
	t.Log(packet)
}

func TestPubcompPacketCodec(t *testing.T) {
	tests := []struct {
		name       string
		encodedHex string
		packet     *PubcompPacket
	}{
		{
			name:       "message 1024",
			encodedHex: "70020400",
			packet: &PubcompPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBCOMP,
					RemainingLength: 2,
				},
				MessageID:         1024,
				PubcompProperties: Properties{RawData: []byte{}},
			},
		},
		{
			name:       "response code 128, no properties",
			encodedHex: "700404008000",
			packet: &PubcompPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBCOMP,
					RemainingLength: 4,
				},
				MessageID:         1024,
				ReasonCode:        0x80,
				PubcompProperties: Properties{RawData: []byte{}},
			},
		},
		{
			name:       "response code 0, with properties",
			encodedHex: "700f0400800b2600036161610003626262",
			packet: &PubcompPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBCOMP,
					RemainingLength: 15,
				},
				MessageID:         1024,
				ReasonCode:        0x80,
				PubcompProperties: Properties{RawData: MustHexDecodeString("2600036161610003626262")},
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
			packet := decoded.(*PubcompPacket)
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
