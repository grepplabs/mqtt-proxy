package config

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/pkg/errors"
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
			Publish              struct {
				Timeout time.Duration
				Async   struct {
					AtMostOnce  bool
					AtLeastOnce bool
					ExactlyOnce bool
				}
			}
		}
		Publisher struct {
			Name string
			Noop struct {
			}
			Kafka struct {
				BootstrapServers string
				GracePeriod      time.Duration
				ConfArgs         KafkaConfigArgs
				DefaultTopic     string
				TopicMappings    TopicMappings
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
			return errors.Errorf("expected key=value, but got %s", pair)
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		if k == "" {
			return errors.Errorf("empty topic key %s", pair)
		}
		if v == "" {
			return errors.Errorf("empty regex value %s", pair)
		}

		r, err := regexp.Compile(v)
		if err != nil {
			return errors.Wrapf(err, "invalid topic mapping regexp '%s'", v)
		}

		c.Mappings = append(c.Mappings, TopicMapping{Topic: k, RegExp: r})
	}
	return nil
}

func (c *TopicMappings) String() string {
	return fmt.Sprintf("%v", c.Mappings)
}
