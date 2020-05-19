package config

import (
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// publisher names
const (
	Noop  = "noop"
	Kafka = "kafka"
)

type Server struct {
	HTTP struct {
		ListenAddress string
		GracePeriod   time.Duration
	}
	MQTT struct {
		ListenAddress    string
		GracePeriod      time.Duration
		ReadTimeout      time.Duration
		WriteTimeout     time.Duration
		IdleTimeout      time.Duration
		ReaderBufferSize int
		WriterBufferSize int
		TLSSrv           struct {
			Cert     string
			Key      string
			ClientCA string
		}
		Handler struct {
			AllowUnauthenticated bool
		}
		Publisher struct {
			Name string
			Noop struct {
			}
			Kafka struct {
				BootstrapServers string
				ConfArgs         KafkaConfigArgs
			}
		}
	}
}

type KafkaConfigArgs struct {
	conf kafka.ConfigMap
}

func (c *KafkaConfigArgs) String() string {
	return fmt.Sprintf("%v", c.conf)
}

func (c *KafkaConfigArgs) Set(value string) error {
	if c.conf == nil {
		c.conf = make(kafka.ConfigMap)
	}
	return c.conf.Set(value)
}
