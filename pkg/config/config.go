package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// publisher names
const (
	PublisherNoop  = "noop"
	PublisherKafka = "kafka"
)

// authenticator names
const (
	AuthNoop  = "noop"
	AuthPlain = "plain"
)

// server certificate source
const (
	CertSourceFile = "file"
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
			Enable     bool
			CertSource string
			Refresh    time.Duration
			File       struct {
				Cert      string
				Key       string
				ClientCA  string
				ClientCLR string
			}
		}
		Handler struct {
			IgnoreUnsupported    []string
			AllowUnauthenticated []string
			Publish              struct {
				Timeout time.Duration
				Async   struct {
					AtMostOnce  bool
					AtLeastOnce bool
					ExactlyOnce bool
				}
			}
			Authenticator struct {
				Name  string
				Plain struct {
					Credentials     map[string]string
					CredentialsFile string
				}
			}
		}
		Publisher struct {
			Name  string
			Kafka struct {
				BootstrapServers string
				GracePeriod      time.Duration
				ConfArgs         KafkaConfigArgs
				DefaultTopic     string
				TopicMappings    TopicMappings
				Workers          int
			}
		}
	}
}

func (c *Server) Init() {
	c.MQTT.Handler.Authenticator.Plain.Credentials = make(map[string]string)
}

func (c Server) Validate() error {
	if c.HTTP.ListenAddress == "" {
		return errors.New("http listen address must not be empty")
	}
	if c.HTTP.GracePeriod < 0 {
		return errors.New("http grace period must be greater than or equal to 0")
	}

	if c.MQTT.ListenAddress == "" {
		return errors.New("mqtt listen address must not be empty")
	}
	if c.MQTT.GracePeriod < 0 {
		return errors.New("mqtt grace period must be greater than or equal to 0")
	}
	if c.MQTT.ReadTimeout < 0 {
		return errors.New("mqtt read timeout must be greater than or equal to 0")
	}
	if c.MQTT.WriteTimeout < 0 {
		return errors.New("mqtt write timeout must be greater than or equal to 0")
	}
	if c.MQTT.IdleTimeout < 0 {
		return errors.New("mqtt idle timeout must be greater than or equal to 0")
	}
	if c.MQTT.ReaderBufferSize < 0 {
		return errors.New("mqtt read buffer size must be greater than or equal to 0")
	}
	if c.MQTT.WriterBufferSize < 0 {
		return errors.New("mqtt write buffer size must be greater than or equal to 0")
	}
	if c.MQTT.Handler.Publish.Timeout < 0 {
		return errors.New("handler publish timeout must be greater than or equal to 0")
	}
	if c.MQTT.Publisher.Name == "" {
		return errors.New("publisher name must not be empty")
	}
	if c.MQTT.Publisher.Name == PublisherKafka {
		if c.MQTT.Publisher.Kafka.BootstrapServers == "" {
			return errors.New("kafka bootstrap servers must not be empty")
		}
		if c.MQTT.Publisher.Kafka.GracePeriod < 0 {
			return errors.New("kafka grace period must be greater than or equal to 0")
		}
		if c.MQTT.Publisher.Kafka.Workers < 1 {
			return errors.New("kafka grace period must be greater than 0")
		}
	}
	return nil
}

type KafkaConfigArgs struct {
	conf kafka.ConfigMap
}

func (c *KafkaConfigArgs) String() string {
	return fmt.Sprintf("%v", c.conf)
}

func (c *KafkaConfigArgs) ConfigMap() kafka.ConfigMap {
	return c.conf
}

func (c *KafkaConfigArgs) Set(value string) error {
	if c.conf == nil {
		c.conf = make(kafka.ConfigMap)
	}
	for _, pair := range strings.Split(value, ",") {
		err := c.conf.Set(pair)
		if err != nil {
			return err
		}
	}
	return nil
}

type TopicMappings struct {
	Mappings []TopicMapping
}

type TopicMapping struct {
	Topic  string
	RegExp *regexp.Regexp
}

func (c *TopicMapping) String() string {
	return fmt.Sprintf("topic=%s regexp=%s", c.Topic, c.RegExp)
}

func (c *TopicMappings) Set(value string) error {
	if c.Mappings == nil {
		c.Mappings = make([]TopicMapping, 0)
	}

	for _, pair := range strings.Split(value, ",") {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 {
			return fmt.Errorf("expected key=value, but got %s", pair)
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		if k == "" {
			return fmt.Errorf("empty topic key %s", pair)
		}
		if v == "" {
			return fmt.Errorf("empty regex value %s", pair)
		}

		r, err := regexp.Compile(v)
		if err != nil {
			return fmt.Errorf("invalid topic mapping regexp '%s': %w", v, err)
		}

		c.Mappings = append(c.Mappings, TopicMapping{Topic: k, RegExp: r})
	}
	return nil
}

func (c *TopicMappings) String() string {
	return fmt.Sprintf("%v", c.Mappings)
}
