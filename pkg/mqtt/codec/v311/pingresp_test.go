package v311

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

func TestNewPingrespPacket(t *testing.T) {
	a := assert.New(t)
	packet := NewControlPacket(mqttproto.PINGRESP).(*PingrespPacket)
	a.Equal(mqttproto.PINGRESP, packet.MessageType)
	a.Equal(mqttproto.MqttMessageTypeNames[packet.MessageType], packet.Name())
	a.Equal(mqttproto.MQTT_3_1_1, packet.Version())
	t.Log(packet)
}

func TestPingrespPacketCodec(t *testing.T) {
	tests := []struct {
		name       string
		encodedHex string
		packet     *PingrespPacket
	}{
		{
			name:       "ping",
			encodedHex: "d000",
			packet: &PingrespPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PINGRESP,
					RemainingLength: 0,
				},
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
			packet := decoded.(*PingrespPacket)
			a.Equal(*tc.packet, *packet)
			a.Equal(mqttproto.MQTT_3_1_1, packet.Version())

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
