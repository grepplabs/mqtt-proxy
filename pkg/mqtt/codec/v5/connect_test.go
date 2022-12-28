package v5

import (
	"bytes"
	"encoding/hex"
	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConnectPacket(t *testing.T) {
	a := assert.New(t)
	packet := NewControlPacket(mqttproto.CONNECT).(*ConnectPacket)
	a.Equal(mqttproto.CONNECT, packet.MessageType)
	a.Equal(mqttproto.MqttMessageTypeNames[packet.MessageType], packet.Name())
	a.Equal(mqttproto.MQTT_5, packet.Version())
	t.Log(packet)
}

func TestDecodeConnectPacket(t *testing.T) {
	tests := []struct {
		name       string
		encodedHex string
		packet     *ConnectPacket
	}{
		{
			name:       "basic connect",
			encodedHex: "101000044d5154540502003c032100140000",
			packet: &ConnectPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.CONNECT,
					RemainingLength: 16,
				},
				ProtocolName:      mqttproto.MQTT,
				ProtocolLevel:     mqttproto.MQTT_5,
				CleanStart:        true,
				KeepAliveSeconds:  60,
				ConnectProperties: Properties{RawData: MustHexDecodeString("210014")},
			},
		},
		{
			name:       "empty will properties",
			encodedHex: "102500044d5154540506003c0321001400000000076d79746f70696300096d796d657373616765",
			packet: &ConnectPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.CONNECT,
					RemainingLength: 37,
				},
				ProtocolName:      mqttproto.MQTT,
				ProtocolLevel:     mqttproto.MQTT_5,
				CleanStart:        true,
				KeepAliveSeconds:  60,
				ConnectProperties: Properties{RawData: MustHexDecodeString("210014")},
				WillFlag:          true,
				WillTopic:         "mytopic",
				WillPayload:       []byte("mymessage"),
				WillProperties:    Properties{RawData: []byte{}},
			},
		},
		{
			name:       "user properties and will properties",
			encodedHex: "103100044d5154540506003c0e260003616161000362626221001400000a0800076d79746f70696300076d79746f7069630000",
			packet: &ConnectPacket{
				FixedHeader: mqttproto.FixedHeader{
					MessageType:     mqttproto.CONNECT,
					RemainingLength: 49,
				},
				ProtocolName:      mqttproto.MQTT,
				ProtocolLevel:     mqttproto.MQTT_5,
				CleanStart:        true,
				KeepAliveSeconds:  60,
				ConnectProperties: Properties{RawData: MustHexDecodeString("2600036161610003626262210014")},
				WillFlag:          true,
				WillTopic:         "mytopic",
				WillPayload:       []byte{},
				WillProperties:    Properties{RawData: MustHexDecodeString("0800076d79746f706963")},
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

func MustHexDecodeString(s string) []byte {
	encodedBytes, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return encodedBytes
}
