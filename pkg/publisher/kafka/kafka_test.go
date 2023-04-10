package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProducerProperties(t *testing.T) {
	tests := []struct {
		name   string
		qos    byte
		input  options
		output kafka.ConfigMap
	}{
		{
			name: "Set producer parameters",
			qos:  0,
			input: options{
				bootstrapServers: "localhost:9092",
				configMap: kafka.ConfigMap{
					"producer.sasl.mechanisms":   "PLAIN",
					"producer.security.protocol": "SASL_SSL",
					"producer.sasl.username":     "myuser",
					"producer.sasl.password":     "mypasswd",
					"{qos-1}.producer.hello":     "property for qos 1",
					"{qos-0}.producer.hello":     "property for qos 0",
				},
			},
			output: kafka.ConfigMap{
				"bootstrap.servers": "localhost:9092",
				"sasl.mechanisms":   "PLAIN",
				"security.protocol": "SASL_SSL",
				"sasl.username":     "myuser",
				"sasl.password":     "mypasswd",
				"hello":             "property for qos 0",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			actual := producerProperties(tc.qos, tc.input)
			a.Equal(tc.output, *actual)
		})
	}
}
