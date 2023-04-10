package config

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
	"time"
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

func TestTopicMappings(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output TopicMappings
		err    error
	}{
		{
			name:  "Set parameters",
			input: "temperature=temperature, humidity=.*/humidity,brightness=.*brightness, temperature=^cool$",
			output: TopicMappings{
				Mappings: []TopicMapping{
					{
						Topic:  "temperature",
						RegExp: regexp.MustCompile(`temperature`),
					},
					{
						Topic:  "humidity",
						RegExp: regexp.MustCompile(`.*/humidity`),
					},
					{
						Topic:  "brightness",
						RegExp: regexp.MustCompile(`.*brightness`),
					},
					{
						Topic:  "temperature",
						RegExp: regexp.MustCompile(`^cool$`),
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			s := new(Server)
			err := s.MQTT.Publisher.Kafka.TopicMappings.Set(tc.input)
			a.Equal(tc.err, err)
			a.Equal(tc.output, s.MQTT.Publisher.Kafka.TopicMappings)
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		factory func() *Server
		err     error
	}{
		{
			name: "noop publisher",
			factory: func() *Server {
				s := new(Server)
				s.HTTP.ListenAddress = "localhost:9090"
				s.MQTT.ListenAddress = "localhost:1883"
				s.MQTT.Publisher.Name = PublisherNoop
				s.MQTT.Publisher.Kafka.Workers = 1
				return s
			},
		},
		{
			name: "kafka publisher",
			factory: func() *Server {
				s := new(Server)
				s.HTTP.ListenAddress = "localhost:9090"
				s.MQTT.ListenAddress = "localhost:1883"
				s.MQTT.Publisher.Name = PublisherKafka
				s.MQTT.Publisher.Kafka.BootstrapServers = "localhost:9092"
				s.MQTT.Publisher.Kafka.Workers = 1
				return s
			},
		},
		{
			name: "kafka publisher with params",
			factory: func() *Server {
				s := new(Server)
				s.HTTP.ListenAddress = "localhost:9090"
				s.HTTP.GracePeriod = 10 * time.Second
				s.MQTT.ListenAddress = "localhost:1883"
				s.MQTT.GracePeriod = 1 * time.Second
				s.MQTT.ReadTimeout = 1 * time.Second
				s.MQTT.WriteTimeout = 1 * time.Second
				s.MQTT.IdleTimeout = 1 * time.Second
				s.MQTT.ReaderBufferSize = 256
				s.MQTT.WriterBufferSize = 256
				s.MQTT.Publisher.Name = PublisherKafka
				s.MQTT.Publisher.Kafka.BootstrapServers = "localhost:9092"
				s.MQTT.Publisher.Kafka.GracePeriod = 10 * time.Second
				s.MQTT.Publisher.Kafka.Workers = 10
				return s
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.factory()
			a := assert.New(t)
			err := input.Validate()
			a.Equal(tc.err, err)
		})
	}
}
