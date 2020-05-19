package mqttcodec

const (
	MQTT            = "MQTT"
	MQTT_3_1_1 byte = 4
	MQTT_5     byte = 5
)

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

const (
	Accepted                           byte = 0
	RefusedUnacceptableProtocolVersion byte = 1
	RefusedIdentifierRejected          byte = 2
	RefusedServerUnavailable           byte = 3
	RefusedBadUserNameOrPassword       byte = 4
	RefusedNotAuthorized               byte = 5
)
