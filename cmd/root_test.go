package cmd

import (
	"github.com/alecthomas/kong"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/grepplabs/mqtt-proxy/pkg/config"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDefaultServerConfig(t *testing.T) {
	testCLI, command, err := parseTestCLI([]string{"server"})
	require.NoError(t, err)
	require.Equal(t, "server", command)
	require.Nil(t, testCLI.Server.MQTT.Handler.Authenticator.Plain.Credentials)
	require.Nil(t, testCLI.Server.MQTT.Publisher.Kafka.ConfArgs.ConfigMap())
	require.Nil(t, testCLI.Server.MQTT.Publisher.Kafka.TopicMappings.Mappings)
	require.Equal(t, 1, testCLI.Server.MQTT.Publisher.Kafka.Workers)
}

func TestKafkaConfigOneParam(t *testing.T) {
	testCLI, _, err := parseTestCLI([]string{
		"server",
		"--mqtt.publisher.kafka.config", "producer.sasl.mechanisms=PLAIN,producer.security.protocol=SASL_SSL,producer.sasl.username=myuser,producer.sasl.password=mypasswd",
	})
	require.NoError(t, err)
	require.EqualValues(t, map[string]kafka.ConfigValue{
		"producer.sasl.mechanisms":   "PLAIN",
		"producer.security.protocol": "SASL_SSL",
		"producer.sasl.username":     "myuser",
		"producer.sasl.password":     "mypasswd",
	}, testCLI.Server.MQTT.Publisher.Kafka.ConfArgs.ConfigMap())
}

func TestKafkaConfigMultipleParams(t *testing.T) {
	testCLI, _, err := parseTestCLI([]string{
		"server",
		"--mqtt.publisher.kafka.config", "producer.sasl.mechanisms=PLAIN,producer.security.protocol=SASL_SSL",
		"--mqtt.publisher.kafka.config", "producer.sasl.username=myuser,producer.sasl.password=mypasswd",
	})
	require.NoError(t, err)
	require.EqualValues(t, map[string]kafka.ConfigValue{
		"producer.sasl.mechanisms":   "PLAIN",
		"producer.security.protocol": "SASL_SSL",
		"producer.sasl.username":     "myuser",
		"producer.sasl.password":     "mypasswd",
	}, testCLI.Server.MQTT.Publisher.Kafka.ConfArgs.ConfigMap())
}

func TestTopicMappingConfig(t *testing.T) {
	testCLI, _, err := parseTestCLI([]string{
		"server",
		"--mqtt.publisher.kafka.topic-mappings", "temperature=temperature, humidity=.*/humidity,brightness=.*brightness, temperature=^cool$",
	})
	require.NoError(t, err)
	require.Equal(t, 4, len(testCLI.Server.MQTT.Publisher.Kafka.TopicMappings.Mappings))
}

func TestPlainCredentialsConfig(t *testing.T) {
	testCLI, _, err := parseTestCLI([]string{
		"server",
		"--mqtt.handler.auth.plain.credentials", "alice=test1",
		"--mqtt.handler.auth.plain.credentials", "bob=test2",
	})
	require.NoError(t, err)
	require.EqualValues(t, map[string]string{
		"alice": "test1",
		"bob":   "test2",
	}, testCLI.Server.MQTT.Handler.Authenticator.Plain.Credentials)
}

func parseTestCLI(args []string) (*CLI, string, error) {
	testCLI := &CLI{}
	parser, err := kong.New(testCLI,
		kong.Name("mqtt-proxy"),
		kong.Description("MQTT Proxy"),
		log.Vars(), config.ServerVars())
	if err != nil {
		return nil, "", err
	}
	command, err := parser.Parse(args)
	if err != nil {
		return nil, "", err
	}
	return testCLI, command.Command(), nil
}
