package config

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/go-playground/validator/v10"
)

// publisher names
const (
	PublisherNoop  = "noop"
	PublisherKafka = "kafka"
	PublisherSQS   = "sqs"
	PublisherSNS   = "sns"
)

// authenticator names
const (
	AuthNoop  = "noop"
	AuthPlain = "plain"
)

// message format
const (
	MessageFormatPlain  = "plain"
	MessageFormatBase64 = "base64"
	MessageFormatJson   = "json"
)

// server certificate source
const (
	CertSourceFile = "file"
)

type Server struct {
	HTTP struct {
		ListenAddress string        `default:"0.0.0.0:9090" help:"Listen host:port for HTTP endpoints." validate:"required"`
		GracePeriod   time.Duration `default:"10s" help:"Time to wait after an interrupt received for HTTP Server." validate:"gte=0"`
	} `embed:"" prefix:"http."`
	MQTT struct {
		ListenAddress    string        `default:"0.0.0.0:1883" help:"Listen host:port for MQTT endpoints." validate:"required"`
		GracePeriod      time.Duration `default:"10s" help:"Time to wait after an interrupt received for MQTT Server." validate:"gte=0"`
		ReadTimeout      time.Duration `default:"5s" help:"Maximum duration for reading the entire request." validate:"gte=0"`
		WriteTimeout     time.Duration `default:"5s" help:"Maximum duration before timing out writes of the response." validate:"gte=0"`
		IdleTimeout      time.Duration `default:"0s" help:"Maximum duration before timing out writes of the response." validate:"gte=0"`
		ReaderBufferSize int           `default:"1024" help:"Read buffer size pro tcp connection." validate:"gte=0"`
		WriterBufferSize int           `default:"1024" help:"Write buffer size pro tcp connection." validate:"gte=0"`
		TLSSrv           struct {
			Enable     bool          `default:"false" help:"Enable server side TLS."`
			CertSource string        `default:"${CertSourceDefault}" enum:"${CertSourceEnum}" help:"TLS certificate source. One of: [${CertSourceEnum}]"`
			Refresh    time.Duration `default:"0s" help:"Option to specify the refresh interval for the TLS certificates." validate:"gte=0"`
			File       struct {
				Cert      string `default:"" help:"TLS Certificate for MQTT server."`
				Key       string `default:"" help:"TLS Key for the MQTT server."`
				ClientCA  string `default:"" help:"TLS CA to verify clients against. If no client CA is specified, there is no client verification on server side."`
				ClientCLR string `default:"" help:"TLS X509 CLR signed be the client CA. If no revocation list is specified, only client CA is verified."`
			} `embed:"" prefix:"file."`
		} `embed:"" prefix:"server-tls."`
		Handler struct {
			IgnoreUnsupported    []string `placeholder:"MSG" enum:"${IgnoreUnsupportedEnum}" help:"List of unsupported messages which are ignored. One of: [${IgnoreUnsupportedEnum}]"`
			AllowUnauthenticated []string `placeholder:"MSG" enum:"${AllowUnauthenticatedEnum}" help:"List of messages for which connection is not disconnected if unauthenticated request is received. One of: [${AllowUnauthenticatedEnum}]"`
			Publish              struct {
				Timeout time.Duration `default:"0s" help:"Maximum duration of sending publish request to broker." validate:"gte=0"`
				Async   struct {
					AtMostOnce  bool `default:"false" help:"Async publish for AT_MOST_ONCE QoS."`
					AtLeastOnce bool `default:"false" help:"Async publish for AT_LEAST_ONCE QoS."`
					ExactlyOnce bool `default:"false" help:"Async publish for EXACTLY_ONCE QoS."`
				} `embed:"" prefix:"async."`
			} `embed:"" prefix:"publish."`
			Authenticator struct {
				Name  string `default:"${AuthDefault}" enum:"${AuthEnum}" help:"Authenticator name. One of: [${AuthEnum}]"`
				Plain struct {
					Credentials     map[string]string `placeholder:"USERNAME=PASSWORD" help:"List of username and password fields."`
					CredentialsFile string            `default:"" help:"Location of a headerless CSV file containing \"usernanme,password\" records."`
				} `embed:"" prefix:"plain."`
			} `embed:"" prefix:"auth."`
		} `embed:"" prefix:"handler."`
		Publisher struct {
			Name          string `default:"${PublisherDefault}" enum:"${PublisherEnum}" help:"Publisher name. One of: [${PublisherEnum}]"`
			MessageFormat string `default:"${MessageFormatDefault}" enum:"${MessageFormatEnum}" help:"Message format. One of: [${MessageFormatEnum}]"`
			Kafka         struct {
				BootstrapServers string          `default:"localhost:9092" help:"Kafka bootstrap servers."`
				GracePeriod      time.Duration   `default:"10s" help:"Time to wait after an interrupt received for Kafka publisher." validate:"gte=0"`
				ConfArgs         KafkaConfigArgs `name:"config" placeholder:"PROP=VAL" help:"Comma separated list of properties."`
				DefaultTopic     string          `default:"" help:"Default Kafka topic for MQTT publish messages."`
				TopicMappings    TopicMappings   `placeholder:"TOPIC=REGEX" help:"Comma separated list of Kafka topic to MQTT topic mappings."`
				Workers          int             `default:"1" help:"Number of kafka publisher workers." validate:"gte=1"`
			} `embed:"" prefix:"kafka."`
			SQS struct {
				AWSProfile    string        `default:"" help:"AWS Profile."`
				AWSRegion     string        `default:"" help:"AWS Region."`
				DefaultQueue  string        `default:"" help:"Default SQS topic for MQTT publish messages."`
				QueueMappings TopicMappings `placeholder:"QUEUE=REGEX" help:"Comma separated list of SQS queue to MQTT topic mappings."`
			} `embed:"" prefix:"sqs."`
			SNS struct {
				AWSProfile       string        `default:"" help:"AWS Profile."`
				AWSRegion        string        `default:"" help:"AWS Region."`
				DefaultTopicARN  string        `default:"" help:"Default topic ARN for MQTT publish messages."`
				TopicARNMappings TopicMappings `placeholder:"TOPIC_ARN=REGEX" help:"Comma separated list of topic ARNs to MQTT topic mappings."`
			} `embed:"" prefix:"sns."`
		} `embed:"" prefix:"publisher."`
	} `embed:"" prefix:"mqtt."`
}

func ServerVars() kong.Vars {
	return map[string]string{
		"CertSourceDefault":        CertSourceFile,
		"CertSourceEnum":           strings.Join([]string{CertSourceFile}, ", "),
		"IgnoreUnsupportedEnum":    strings.Join([]string{"SUBSCRIBE", "UNSUBSCRIBE"}, ", "),
		"AllowUnauthenticatedEnum": strings.Join([]string{"PUBLISH", "PUBREL", "PINGREQ"}, ", "),
		"AuthDefault":              AuthNoop,
		"AuthEnum":                 strings.Join([]string{AuthNoop, AuthPlain}, ", "),
		"PublisherDefault":         PublisherNoop,
		"PublisherEnum":            strings.Join([]string{PublisherNoop, PublisherKafka, PublisherSQS, PublisherSNS}, ", "),
		"MessageFormatDefault":     MessageFormatPlain,
		"MessageFormatEnum":        strings.Join([]string{MessageFormatPlain, MessageFormatBase64, MessageFormatJson}, ", "),
	}
}

func (c *Server) Init() {
	c.MQTT.Handler.Authenticator.Plain.Credentials = make(map[string]string)
}

func (c *Server) Validate() error {
	validate := validator.New()
	err := validate.Struct(c)
	if err != nil {
		return fmt.Errorf("config validation failure: %w", err)
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

// UnmarshalText implements Kong encoding.TextUnmarshaler
func (c *KafkaConfigArgs) UnmarshalText(text []byte) error {
	return c.Set(string(text))
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

// UnmarshalText implements Kong encoding.TextUnmarshaler
func (c *TopicMappings) UnmarshalText(text []byte) error {
	return c.Set(string(text))
}
