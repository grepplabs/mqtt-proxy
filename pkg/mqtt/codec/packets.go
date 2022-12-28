package codec

import (
	"bytes"
	"fmt"
	"io"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
	mqtt311 "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/v311"
	mqtt5 "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/v5"
)

func ReadPacket(r io.Reader, protocolVersion byte) (mqttproto.ControlPacket, error) {
	if protocolVersion == 0 {
		var (
			err error
			buf bytes.Buffer
		)
		versionReader := io.TeeReader(r, &buf)
		protocolVersion, err = readConnectVersion(versionReader)
		if err != nil {
			return nil, err
		}
		r = io.MultiReader(bytes.NewReader(buf.Bytes()), r)
	}
	switch protocolVersion {
	case mqttproto.MQTT_3_1_1:
		return mqtt311.ReadPacket(r)
	case mqttproto.MQTT_5:
		return mqtt5.ReadPacket(r)
	default:
		return nil, mqtt311.NewConnAckError(mqttproto.RefusedUnacceptableProtocolVersion, fmt.Sprintf("unsupported protocol version %v", protocolVersion))
	}
}

func readConnectVersion(r io.Reader) (byte, error) {
	// fixed header
	var fh mqttproto.FixedHeader
	b1 := make([]byte, 1)
	_, err := io.ReadFull(r, b1)
	if err != nil {
		return 0, err
	}
	err = fh.Unpack(b1[0], r)
	if err != nil {
		return 0, err
	}
	err = fh.Validate()
	if err != nil {
		return 0, err
	}
	if fh.MessageType != mqttproto.CONNECT {
		return 0, fmt.Errorf("expected CONNECT packet but got type 0x%x", fh.MessageType)
	}
	// variable header
	// Protocol Name
	_, err = mqttproto.DecodeString(r)
	if err != nil {
		return 0, err
	}
	// Protocol Version
	version, err := mqttproto.DecodeByte(r)
	if err != nil {
		return 0, err
	}
	return version, nil
}
