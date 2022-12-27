package proto

const (
	MQTT                               = "MQTT"
	MQTT_3_1_1                    byte = 4
	MQTT_5                        byte = 5
	MQTT_DEFAULT_PROTOCOL_VERSION      = MQTT_3_1_1
)

var MqttProtocolVersionNames = map[byte]string{
	MQTT_3_1_1: "3.1.1",
	MQTT_5:     "5",
}

func MqttProtocolVersionName(version byte) string {
	return MqttProtocolVersionNames[version]
}

const (
	CONNECT     byte = 1
	CONNACK     byte = 2
	PUBLISH     byte = 3
	PUBACK      byte = 4
	PUBREC      byte = 5
	PUBREL      byte = 6
	PUBCOMP     byte = 7
	SUBSCRIBE   byte = 8
	SUBACK      byte = 9
	UNSUBSCRIBE byte = 10
	UNSUBACK    byte = 11
	PINGREQ     byte = 12
	PINGRESP    byte = 13
	DISCONNECT  byte = 14
	AUTH        byte = 15 // mqtt-v5.0
)

var MqttMessageTypeNames = map[byte]string{
	CONNECT:     "CONNECT",
	CONNACK:     "CONNACK",
	PUBLISH:     "PUBLISH",
	PUBACK:      "PUBACK",
	PUBREC:      "PUBREC",
	PUBREL:      "PUBREL",
	PUBCOMP:     "PUBCOMP",
	SUBSCRIBE:   "SUBSCRIBE",
	SUBACK:      "SUBACK",
	UNSUBSCRIBE: "UNSUBSCRIBE",
	UNSUBACK:    "UNSUBACK",
	PINGREQ:     "PINGREQ",
	PINGRESP:    "PINGRESP",
	DISCONNECT:  "DISCONNECT",
	AUTH:        "AUTH",
}

const (
	AT_MOST_ONCE  = 0
	AT_LEAST_ONCE = 1
	EXACTLY_ONCE  = 2
	FAILURE       = 0x80
)

var MqttQoSNames = map[byte]string{
	AT_MOST_ONCE:  "AT_MOST_ONCE",
	AT_LEAST_ONCE: "AT_LEAST_ONCE",
	EXACTLY_ONCE:  "EXACTLY_ONCE",
	FAILURE:       "FAILURE",
}

// MQTT 3.1.1 - 3.2.2.3 Connect Return code
const (
	Accepted                           byte = 0
	RefusedUnacceptableProtocolVersion byte = 1
	RefusedIdentifierRejected          byte = 2
	RefusedServerUnavailable           byte = 3
	RefusedBadUserNameOrPassword       byte = 4
	RefusedNotAuthorized               byte = 5
)

// MQTT 5 - 3.2.2.2 Connect Reason Code
const (
	RefusedUnspecifiedError           byte = 0x80 // 128
	RefusedUnsupportedProtocolVersion byte = 0x84 // 132
)
