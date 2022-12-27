package v311

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

func TestNewPublishPacket(t *testing.T) {
	a := assert.New(t)
	packet := NewControlPacket(mqttproto.PUBLISH).(*PublishPacket)
	a.Equal(mqttproto.PUBLISH, packet.MessageType)
	a.Equal(mqttproto.MqttMessageTypeNames[packet.MessageType], packet.Name())
	t.Log(packet)
}

func TestPublishPacketCodec(t *testing.T) {
	tests := []struct {
		name       string
		encodedHex string
		packet     *PublishPacket
	}{
		{
			name:       "QoS 0",
			encodedHex: "3012000564756d6d7948656c6c6f20776f726c64",
			packet: &PublishPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBLISH,
					Qos:             mqttproto.AT_MOST_ONCE,
					RemainingLength: 18,
				},
				TopicName: "dummy",
				Message:   []byte("Hello world"),
			},
		},
		{
			name:       "QoS 0, empty message",
			encodedHex: "3007000564756d6d79",
			packet: &PublishPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBLISH,
					Qos:             mqttproto.AT_MOST_ONCE,
					RemainingLength: 7,
				},
				TopicName: "dummy",
				Message:   []byte{},
			},
		},
		{
			name:       "QoS 1",
			encodedHex: "320b000564756d6d7900016f6e",
			packet: &PublishPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBLISH,
					Qos:             mqttproto.AT_LEAST_ONCE,
					RemainingLength: 11,
				},
				TopicName: "dummy",
				MessageID: 1,
				Message:   []byte("on"),
			},
		},
		{
			name:       "QoS 1, empty message",
			encodedHex: "3209000564756d6d790001",
			packet: &PublishPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBLISH,
					Qos:             mqttproto.AT_LEAST_ONCE,
					RemainingLength: 9,
				},
				TopicName: "dummy",
				MessageID: 1,
				Message:   []byte{},
			},
		},
		{
			name:       "QoS 2",
			encodedHex: "340b000564756d6d7900016f6e",
			packet: &PublishPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBLISH,
					Qos:             mqttproto.EXACTLY_ONCE,
					RemainingLength: 11,
				},
				TopicName: "dummy",
				MessageID: 1,
				Message:   []byte("on"),
			},
		},
		{
			name:       "QoS 2, empty message",
			encodedHex: "3409000564756d6d790001",
			packet: &PublishPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBLISH,
					Qos:             mqttproto.EXACTLY_ONCE,
					RemainingLength: 9,
				},
				TopicName: "dummy",
				MessageID: 1,
				Message:   []byte{},
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
			packet := decoded.(*PublishPacket)
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
