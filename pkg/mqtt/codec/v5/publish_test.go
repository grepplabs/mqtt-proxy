package v5

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
	a.Equal(mqttproto.MQTT_5, packet.Version())
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
			encodedHex: "3019000564756d6d790048656c6c6f20776f726c6420716f732030",
			packet: &PublishPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBLISH,
					Qos:             mqttproto.AT_MOST_ONCE,
					RemainingLength: 25,
				},
				TopicName:         "dummy",
				Message:           []byte("Hello world qos 0"),
				PublishProperties: Properties{RawData: []byte{}},
			},
		},
		{
			name:       "QoS 0 with props",
			encodedHex: "301b000564756d6d79112600056d796b657900076d7976616c75656f6e",
			packet: &PublishPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBLISH,
					Qos:             mqttproto.AT_MOST_ONCE,
					RemainingLength: 27,
				},
				TopicName:         "dummy",
				Message:           []byte("on"),
				PublishProperties: Properties{RawData: MustHexDecodeString("2600056d796b657900076d7976616c7565")},
			},
		},
		{
			name:       "QoS 0, empty message",
			encodedHex: "3008000564756d6d7900",
			packet: &PublishPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBLISH,
					Qos:             mqttproto.AT_MOST_ONCE,
					RemainingLength: 8,
				},
				TopicName:         "dummy",
				Message:           []byte{},
				PublishProperties: Properties{RawData: []byte{}},
			},
		},
		{
			name:       "QoS 1",
			encodedHex: "320c000564756d6d790001006f6e",
			packet: &PublishPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBLISH,
					Qos:             mqttproto.AT_LEAST_ONCE,
					RemainingLength: 12,
				},
				TopicName:         "dummy",
				MessageID:         1,
				Message:           []byte("on"),
				PublishProperties: Properties{RawData: []byte{}},
			},
		},
		{
			name:       "QoS 1 with props",
			encodedHex: "321d000564756d6d790001112600056d796b657900076d7976616c75656f6e",
			packet: &PublishPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBLISH,
					Qos:             mqttproto.AT_LEAST_ONCE,
					RemainingLength: 29,
				},
				TopicName:         "dummy",
				MessageID:         1,
				Message:           []byte("on"),
				PublishProperties: Properties{RawData: MustHexDecodeString("2600056d796b657900076d7976616c7565")},
			},
		},
		{
			name:       "QoS 1, empty message",
			encodedHex: "320a000564756d6d79000100",
			packet: &PublishPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBLISH,
					Qos:             mqttproto.AT_LEAST_ONCE,
					RemainingLength: 10,
				},
				TopicName:         "dummy",
				MessageID:         1,
				Message:           []byte{},
				PublishProperties: Properties{RawData: []byte{}},
			},
		},
		{
			name:       "QoS 2",
			encodedHex: "340c000564756d6d790001006f6e",
			packet: &PublishPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBLISH,
					Qos:             mqttproto.EXACTLY_ONCE,
					RemainingLength: 12,
				},
				TopicName:         "dummy",
				MessageID:         1,
				Message:           []byte("on"),
				PublishProperties: Properties{RawData: []byte{}},
			},
		},

		{
			name:       "QoS 2, empty message",
			encodedHex: "340a000564756d6d79000100",
			packet: &PublishPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBLISH,
					Qos:             mqttproto.EXACTLY_ONCE,
					RemainingLength: 10,
				},
				TopicName:         "dummy",
				MessageID:         1,
				Message:           []byte{},
				PublishProperties: Properties{RawData: []byte{}},
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
