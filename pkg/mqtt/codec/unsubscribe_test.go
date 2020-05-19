package mqttcodec

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUnsubscribePacket(t *testing.T) {
	a := assert.New(t)
	packet := NewControlPacket(UNSUBSCRIBE).(*UnsubscribePacket)
	a.Equal(UNSUBSCRIBE, packet.MessageType)
	a.Equal(MqttMessageTypeNames[packet.MessageType], packet.Name())
	t.Log(packet)
}

func TestUnsubscribePacketCodec(t *testing.T) {

	newPacket := func(msgLen int, messageID uint16, topicFilters ...string) *UnsubscribePacket {
		return &UnsubscribePacket{
			FixedHeader: FixedHeader{
				MessageType:     UNSUBSCRIBE,
				Qos:             AT_LEAST_ONCE,
				RemainingLength: msgLen,
			},
			MessageID:    messageID,
			TopicFilters: topicFilters,
		}
	}

	tests := []struct {
		name       string
		encodedHex string
		packet     *UnsubscribePacket
	}{
		{
			name:       "unsubscribe",
			packet:     newPacket(7, 2, "a/b"),
			encodedHex: "a20700020003612f62",
		},
		{
			name:       "unsubscribe 2 topics",
			packet:     newPacket(12, 1, "a/b", "c/d"),
			encodedHex: "a20c00010003612f620003632f64",
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
			packet := decoded.(*UnsubscribePacket)
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
