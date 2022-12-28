package v5

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

func TestNewDisconnectPacket(t *testing.T) {
	a := assert.New(t)
	packet := NewControlPacket(mqttproto.DISCONNECT).(*DisconnectPacket)
	a.Equal(mqttproto.DISCONNECT, packet.MessageType)
	a.Equal(mqttproto.MqttMessageTypeNames[packet.MessageType], packet.Name())
	a.Equal(mqttproto.MQTT_5, packet.Version())
	t.Log(packet)
}

func TestDisconnectPacketCodec(t *testing.T) {
	tests := []struct {
		name       string
		encodedHex string
		packet     *DisconnectPacket
	}{
		{
			name:       "disconnect reason code 0, no properties",
			encodedHex: "e000",
			packet: &DisconnectPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.DISCONNECT,
					RemainingLength: 0,
				},
				ReasonCode:           0,
				DisconnectProperties: Properties{RawData: []byte{}},
			},
		},
		{
			name:       "disconnect unspecified error, no properties",
			encodedHex: "e0028000",
			packet: &DisconnectPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.DISCONNECT,
					RemainingLength: 2,
				},
				ReasonCode:           0x80,
				DisconnectProperties: Properties{RawData: []byte{}},
			},
		},
		{
			name:       "disconnect reason code 0, with properties",
			encodedHex: "e00d000b2600036161610003626262",
			packet: &DisconnectPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.DISCONNECT,
					RemainingLength: 13,
				},
				ReasonCode:           0,
				DisconnectProperties: Properties{RawData: MustHexDecodeString("2600036161610003626262")},
			},
		},
		{
			name:       "disconnect reason code 0, with properties",
			encodedHex: "e01380112600056d796b657900076d7976616c7565",
			packet: &DisconnectPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.DISCONNECT,
					RemainingLength: 19,
				},
				ReasonCode:           0x80,
				DisconnectProperties: Properties{RawData: MustHexDecodeString("2600056d796b657900076d7976616c7565")},
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
			packet := decoded.(*DisconnectPacket)
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
