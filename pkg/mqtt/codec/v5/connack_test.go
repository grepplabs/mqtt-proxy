package v5

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

func TestNewConnackPacket(t *testing.T) {
	a := assert.New(t)
	packet := NewControlPacket(mqttproto.CONNACK).(*ConnackPacket)
	a.Equal(mqttproto.CONNACK, packet.MessageType)
	a.Equal(mqttproto.MqttMessageTypeNames[packet.MessageType], packet.Name())
	a.Equal(mqttproto.MQTT_5, packet.Version())
	t.Log(packet)
}

func TestConnackPacketCodec(t *testing.T) {
	tests := []struct {
		name       string
		encodedHex string
		packet     *ConnackPacket
	}{
		{
			name:       "connection accepted",
			encodedHex: "2003000000",
			packet: &ConnackPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.CONNACK,
					RemainingLength: 3,
				},
				SessionPresent:    false,
				ReturnCode:        0,
				ConnackProperties: Properties{RawData: []byte{}},
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
			packet := decoded.(*ConnackPacket)
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
