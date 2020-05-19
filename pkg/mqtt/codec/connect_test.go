package mqttcodec

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConnectPacket(t *testing.T) {
	a := assert.New(t)
	packet := NewControlPacket(CONNECT).(*ConnectPacket)
	a.Equal(CONNECT, packet.MessageType)
	a.Equal(MqttMessageTypeNames[packet.MessageType], packet.Name())
	t.Log(packet)
}

func TestDecodeConnectPacket(t *testing.T) {
	tests := []struct {
		name       string
		encodedHex string
		packet     *ConnectPacket
	}{
		{
			name:       "Will QOS - At most once delivery",
			encodedHex: "102300044d5154540402003c00176d6f73712d5073513573716431327175776c3530735932",
			packet: &ConnectPacket{
				FixedHeader: FixedHeader{
					MessageType:     CONNECT,
					RemainingLength: 35,
				},
				ProtocolName:     MQTT,
				ProtocolLevel:    MQTT_3_1_1,
				CleanSession:     true,
				KeepAliveSeconds: 60,
				ClientIdentifier: "mosq-PsQ5sqd12quwl50sY2",
			},
		},
		{
			name:       "Will QOS - At least once delivery",
			encodedHex: "103a00044d515454040e003c00176d6f73712d4a426a3275354f67587a52786978666a6278001373656e736f72732f74656d70657261747572650000",
			packet: &ConnectPacket{
				FixedHeader: FixedHeader{
					MessageType:     CONNECT,
					RemainingLength: 58,
				},
				ProtocolName:     MQTT,
				ProtocolLevel:    MQTT_3_1_1,
				CleanSession:     true,
				WillFlag:         true,
				WillQos:          AT_LEAST_ONCE,
				KeepAliveSeconds: 60,
				ClientIdentifier: "mosq-JBj2u5OgXzRxixfjbx",
				WillTopic:        "sensors/temperature",
				WillMessage:      []byte{},
			},
		},
		{
			name:       "Will QOS - Exactly once delivery",
			encodedHex: "105800044d5154540434007800116d7174742d70726f78792e636c69656e74001e73776974636865732f6b69746368656e5f6c69676874732f7374617475730019646973636f6e6e656374656420756e65787065637465646c79",
			packet: &ConnectPacket{
				FixedHeader: FixedHeader{
					MessageType:     CONNECT,
					RemainingLength: 88,
				},
				ProtocolName:     MQTT,
				ProtocolLevel:    MQTT_3_1_1,
				CleanSession:     false,
				WillFlag:         true,
				WillRetain:       true,
				WillQos:          EXACTLY_ONCE,
				KeepAliveSeconds: 120,
				ClientIdentifier: "mqtt-proxy.client",
				WillTopic:        "switches/kitchen_lights/status",
				WillMessage:      []byte("disconnected unexpectedly"),
			},
		},
		{
			name:       "Username / Password",
			encodedHex: "103300044d51545404c2003c00116d7174742d70726f78792e636c69656e7400076d792d75736572000b6d792d70617373776f7264",
			packet: &ConnectPacket{
				FixedHeader: FixedHeader{
					MessageType:     CONNECT,
					RemainingLength: 51,
				},
				ProtocolName:     MQTT,
				ProtocolLevel:    MQTT_3_1_1,
				HasUsername:      true,
				HasPassword:      true,
				CleanSession:     true,
				KeepAliveSeconds: 60,
				ClientIdentifier: "mqtt-proxy.client",
				Username:         "my-user",
				Password:         []byte("my-password"),
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
			packet := decoded.(*ConnectPacket)
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
