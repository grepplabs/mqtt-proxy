package mqttcodec

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSubscribePacket(t *testing.T) {
	a := assert.New(t)
	packet := NewControlPacket(SUBSCRIBE).(*SubscribePacket)
	a.Equal(SUBSCRIBE, packet.MessageType)
	a.Equal(MqttMessageTypeNames[packet.MessageType], packet.Name())
	t.Log(packet)
}

func TestSubscribePacketCodec(t *testing.T) {

	newPacket := func(msgLen int, messageID uint16, topicSubscriptions ...TopicSubscription) *SubscribePacket {
		return &SubscribePacket{
			FixedHeader: FixedHeader{
				MessageType:     SUBSCRIBE,
				Qos:             AT_LEAST_ONCE,
				RemainingLength: msgLen,
			},
			MessageID:          messageID,
			TopicSubscriptions: topicSubscriptions,
		}
	}

	tests := []struct {
		name       string
		encodedHex string
		packet     *SubscribePacket
	}{
		{
			name: "subscribe qos 0",
			packet: newPacket(10, 1, TopicSubscription{
				TopicFilter: "dummy",
				Qos:         AT_MOST_ONCE,
			}),
			encodedHex: "820a0001000564756d6d7900",
		},
		{
			name: "subscribe qos 1",
			packet: newPacket(8, 1, TopicSubscription{
				TopicFilter: "a/b",
				Qos:         AT_LEAST_ONCE,
			}),
			encodedHex: "820800010003612f6201",
		},
		{
			name: "subscribe qos 2",
			packet: newPacket(8, 1, TopicSubscription{
				TopicFilter: "c/d",
				Qos:         EXACTLY_ONCE,
			}),
			encodedHex: "820800010003632f6402",
		},
		{
			name: "multiple subscriptions qos 1",
			packet: newPacket(14, 1,
				TopicSubscription{
					TopicFilter: "a/b",
					Qos:         AT_LEAST_ONCE,
				},
				TopicSubscription{
					TopicFilter: "c/d",
					Qos:         AT_LEAST_ONCE,
				}),
			encodedHex: "820e00010003612f62010003632f6401",
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
			packet := decoded.(*SubscribePacket)
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
