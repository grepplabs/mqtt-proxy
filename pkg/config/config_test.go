package config

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKafkaConfArgs(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output KafkaConfigArgs
		err    error
	}{
		{
			name:  "Set parameters",
			input: "bootstrap.servers=localhost:9092,producer.sasl.mechanisms=PLAIN,producer.security.protocol=SASL_SSL,producer.sasl.username=myuser,producer.sasl.password=mypasswd,{qos-0}.producer.hello=property for qos 0",
			output: KafkaConfigArgs{
				conf: map[string]kafka.ConfigValue{
					"bootstrap.servers":          "localhost:9092",
					"producer.sasl.mechanisms":   "PLAIN",
					"producer.security.protocol": "SASL_SSL",
					"producer.sasl.username":     "myuser",
					"producer.sasl.password":     "mypasswd",
					"{qos-0}.producer.hello":     "property for qos 0",
				},
			},
		},
		{
			name:  "Error expected key=value",
			input: "bootstrap.servers",
			output: KafkaConfigArgs{
				conf: map[string]kafka.ConfigValue{},
			},
			err: kafka.NewError(-186, "Expected key=value", false),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			s := new(Server)
			err := s.MQTT.Publisher.Kafka.ConfArgs.Set(tc.input)
			a.Equal(tc.err, err)
			a.Equal(tc.output, s.MQTT.Publisher.Kafka.ConfArgs)
		})
	}
}
