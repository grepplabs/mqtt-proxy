package mqttcodec

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConnackPacket(t *testing.T) {
	a := assert.New(t)
	packet := NewControlPacket(CONNACK).(*ConnackPacket)
	a.Equal(CONNACK, packet.MessageType)
	a.Equal(MqttMessageTypeNames[packet.MessageType], packet.Name())
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
			encodedHex: "20020000",
			packet: &ConnackPacket{
				FixedHeader: FixedHeader{
					MessageType:     CONNACK,
					RemainingLength: 2,
				},
				SessionPresent: false,
				ReturnCode:     0,
			},
		},
		{
			name:       "connection refused",
			encodedHex: "20020004",
			packet: &ConnackPacket{
				FixedHeader: FixedHeader{
					MessageType:     CONNACK,
					RemainingLength: 2,
				},
				SessionPresent: false,
				ReturnCode:     RefusedBadUserNameOrPassword,
			},
		},
		{
			name:       "session accepted, session present",
			encodedHex: "20020100",
			packet: &ConnackPacket{
				FixedHeader: FixedHeader{
					MessageType:     CONNACK,
					RemainingLength: 2,
				},
				SessionPresent: true,
				ReturnCode:     0,
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
