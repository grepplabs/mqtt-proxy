package codec

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
	mqtt311 "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/v311"
)

func TestReadPacketV311(t *testing.T) {
	tests := []struct {
		name       string
		encodedHex string
		packet     mqttproto.ControlPacket
	}{
		{
			name:       "connect",
			encodedHex: "102300044d5154540402003c00176d6f73712d5073513573716431327175776c3530735932",
			packet: &mqtt311.ConnectPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.CONNECT,
					RemainingLength: 35,
				},
				ProtocolName:     mqttproto.MQTT,
				ProtocolLevel:    mqttproto.MQTT_3_1_1,
				CleanSession:     true,
				KeepAliveSeconds: 60,
				ClientIdentifier: "mosq-PsQ5sqd12quwl50sY2",
			},
		},
		{
			name:       "connack",
			encodedHex: "20020000",
			packet: &mqtt311.ConnackPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.CONNACK,
					RemainingLength: 2,
				},
				SessionPresent: false,
				ReturnCode:     0,
			},
		},
		{
			name:       "disconnect",
			encodedHex: "e000",
			packet: &mqtt311.DisconnectPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.DISCONNECT,
					RemainingLength: 0,
				},
			},
		},
		{
			name:       "ping req",
			encodedHex: "c000",
			packet: &mqtt311.PingreqPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PINGREQ,
					RemainingLength: 0,
				},
			},
		},
		{
			name:       "ping resp",
			encodedHex: "d000",
			packet: &mqtt311.PingrespPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PINGRESP,
					RemainingLength: 0,
				},
			},
		},
		{
			name:       "pub ack",
			encodedHex: "40020001",
			packet: &mqtt311.PubackPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBACK,
					RemainingLength: 2,
				},
				MessageID: 1,
			},
		},
		{
			name:       "pub comp",
			encodedHex: "70020001",
			packet: &mqtt311.PubcompPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBCOMP,
					RemainingLength: 2,
				},
				MessageID: 1,
			},
		},
		{
			name:       "publish",
			encodedHex: "3012000564756d6d7948656c6c6f20776f726c64",
			packet: &mqtt311.PublishPacket{
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
			name:       "pub rec",
			encodedHex: "50020001",
			packet: &mqtt311.PubrecPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBREC,
					RemainingLength: 2,
				},
				MessageID: 1,
			},
		},
		{
			name:       "pub rel",
			encodedHex: "62020001",
			packet: &mqtt311.PubrelPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.PUBREL,
					Qos:             mqttproto.AT_LEAST_ONCE,
					RemainingLength: 2,
				},
				MessageID: 1,
			},
		},
		{
			name:       "sub ack",
			encodedHex: "9003000101",
			packet: &mqtt311.SubackPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.SUBACK,
					RemainingLength: 3,
				},
				MessageID:   1,
				ReturnCodes: []byte{mqttproto.AT_LEAST_ONCE},
			},
		},
		{
			name: "subscribe",
			packet: &mqtt311.SubscribePacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.SUBSCRIBE,
					Qos:             mqttproto.AT_LEAST_ONCE,
					RemainingLength: 10,
				},
				MessageID: 1,
				TopicSubscriptions: []mqtt311.TopicSubscription{
					{
						TopicFilter: "dummy",
						Qos:         mqttproto.AT_MOST_ONCE,
					},
				},
			},
			encodedHex: "820a0001000564756d6d7900",
		},
		{
			name:       "subscribe ack",
			encodedHex: "b0020002",
			packet: &mqtt311.UnsubackPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.UNSUBACK,
					RemainingLength: 2,
				},
				MessageID: 2,
			},
		},
		{
			name: "unsubscribe",
			packet: &mqtt311.UnsubscribePacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.UNSUBSCRIBE,
					Qos:             mqttproto.AT_LEAST_ONCE,
					RemainingLength: 7,
				},
				MessageID:    2,
				TopicFilters: []string{"a/b"},
			},
			encodedHex: "a20700020003612f62",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Log(tc.packet)
			// decode
			encodedBytes, err := hex.DecodeString(tc.encodedHex)
			require.Nil(t, err)
			// read connect version and decode
			decoded, err := ReadPacket(bytes.NewReader(encodedBytes), 0)
			if tc.packet.Type() == mqttproto.CONNECT {
				require.Nil(t, err)
				require.Equal(t, tc.packet, decoded)
			} else {
				require.NotNil(t, err, "CONNECT is required for protocolVersion eq 0")
			}
			// decode only
			decoded, err = ReadPacket(bytes.NewReader(encodedBytes), mqttproto.MQTT_3_1_1)
			require.Nil(t, err)
			require.Equal(t, tc.packet, decoded)
			require.Equal(t, mqttproto.MQTT_3_1_1, decoded.Version())
		})
	}
}
