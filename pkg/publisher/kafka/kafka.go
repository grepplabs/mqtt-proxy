package kafka

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	mqttcodec "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec"
	"github.com/grepplabs/mqtt-proxy/pkg/runtime"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/atomic"
)

const (
	mqttQosHeader    = "mqtt.qos"
	mqttDupHeader    = "mqtt.dup"
	mqttRetainHeader = "mqtt.retain"
	mqttMsgIDHeader  = "mqtt.packet.id"
)
const (
	shutdownPollInterval = 500 * time.Millisecond
)

type kafkaProducer struct {
	*kafka.Producer
	logger    log.Logger
	closeOnce sync.Once
}

func (k *kafkaProducer) Close() {
	k.closeOnce.Do(func() {
		k.Producer.Close()
	})
}

type Publisher struct {
	logger log.Logger

	producers map[byte]*kafkaProducer

	inShutdown atomic.Bool

	workersDone *runtime.DoneChannel

	opts options
}

func New(logger log.Logger, registry *prometheus.Registry, opts ...Option) (*Publisher, error) {
	options := options{workers: 1}
	for _, o := range opts {
		o.apply(&options)
	}
	err := options.validate()
	if err != nil {
		return nil, err
	}
	producers, err := newProducers(logger, options)
	if err != nil {
		return nil, err
	}

	publisher := &Publisher{
		logger:      logger,
		producers:   producers,
		workersDone: runtime.NewDoneChannel(),
		opts:        options,
	}
	return publisher, nil
}

func (s *Publisher) flush(timeoutMs int) int {
	var wg sync.WaitGroup
	wg.Add(len(s.producers))

	var remaining atomic.Int32
	for _, producer := range s.producers {
		go func(p *kafkaProducer) {
			defer wg.Done()

			p.logger.Debugf("flush kafka producer and wait max %v", timeoutMs)

			remain := p.Flush(timeoutMs)
			remaining.Add(int32(remain))
		}(producer)
	}
	wg.Wait()

	return int(remaining.Load())
}

func (s *Publisher) Shutdown(err error) {
	defer s.logger.WithError(err).Infof("internal server shutdown")

	if s.opts.gracePeriod == 0 {
		_ = s.Close()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.opts.gracePeriod)
	defer cancel()

	if err := s.shutdown(ctx); err != nil {
		s.logger.WithError(err).Infof("internal server shut down failed")
	}
	return
}

func (s *Publisher) shutdown(ctx context.Context) error {
	if !s.inShutdown.CAS(false, true) {
		return nil
	}
	defer s.Close()

	ticker := time.NewTicker(shutdownPollInterval)

	var remain int
	defer ticker.Stop()
	for {
		remain = s.flush(int(shutdownPollInterval.Milliseconds()))
		if remain == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Publisher) Close() error {
	s.inShutdown.Store(true)

	defer s.logger.Infof("kafka publisher closed")

	s.workersDone.Close()

	for _, producer := range s.producers {
		producer.Close()
	}
	return nil
}

func newProducers(logger log.Logger, opts options) (map[byte]*kafkaProducer, error) {
	producers := make(map[byte]*kafkaProducer)

	atMostOnceProps := producerProperties(mqttcodec.AT_MOST_ONCE, opts)
	_ = atMostOnceProps.SetKey("acks", "0")
	atMostOnceProducer, err := kafka.NewProducer(atMostOnceProps)
	if err != nil {
		return nil, err
	}
	producers[mqttcodec.AT_MOST_ONCE] = &kafkaProducer{
		Producer: atMostOnceProducer, logger: logger.WithField("qos", "0")}

	atLeastOnceProps := producerProperties(mqttcodec.AT_LEAST_ONCE, opts)
	_ = atLeastOnceProps.SetKey("acks", "all")
	atLeastOnceProducer, err := kafka.NewProducer(atLeastOnceProps)
	if err != nil {
		closeProducers(producers)
		return nil, err
	}
	producers[mqttcodec.AT_LEAST_ONCE] = &kafkaProducer{
		Producer: atLeastOnceProducer, logger: logger.WithField("qos", "1")}

	exactlyOnceProps := producerProperties(mqttcodec.EXACTLY_ONCE, opts)
	_ = exactlyOnceProps.SetKey("acks", "all")
	_ = exactlyOnceProps.SetKey("enable.idempotence", "true")
	exactlyOnceProducer, err := kafka.NewProducer(exactlyOnceProps)
	if err != nil {
		closeProducers(producers)
		return nil, err
	}
	producers[mqttcodec.EXACTLY_ONCE] = &kafkaProducer{
		Producer: exactlyOnceProducer, logger: logger.WithField("qos", "2")}

	return producers, nil
}

func closeProducers(producers map[byte]*kafkaProducer) {
	for _, p := range producers {
		p.Close()
	}
}

func producerProperties(qos byte, opts options) *kafka.ConfigMap {
	configMap := make(kafka.ConfigMap)
	_ = configMap.SetKey("bootstrap.servers", opts.bootstrapServers)
	for k, v := range propertiesWithPrefix(opts.configMap, "producer.", true) {
		_ = configMap.SetKey(k, v)
	}
	for k, v := range propertiesWithPrefix(opts.configMap, fmt.Sprintf("{qos=%d}.producer.", qos), true) {
		_ = configMap.SetKey(k, v)
	}
	return &configMap
}

func propertiesWithPrefix(config kafka.ConfigMap, prefix string, strip bool) kafka.ConfigMap {
	result := make(kafka.ConfigMap)
	for k, v := range config {
		if strings.HasPrefix(k, prefix) {
			if strip {
				_ = result.SetKey(strings.TrimPrefix(k, prefix), v)
			} else {
				_ = result.SetKey(k, v)
			}
		}
	}
	return result
}

func (s *Publisher) newKafkaMessage(req *apis.PublishRequest, opaque interface{}) (*kafka.Message, error) {

	//TODO: map mqtt topic to kafka topic
	kafkaTopic := "myTopic"

	headers := []kafka.Header{
		{Key: mqttQosHeader, Value: []byte(strconv.FormatUint(uint64(req.Qos), 10))},
		{Key: mqttDupHeader, Value: []byte(strconv.FormatBool(req.Dup))},
		{Key: mqttRetainHeader, Value: []byte(strconv.FormatBool(req.Retain))},
		{Key: mqttMsgIDHeader, Value: []byte(strconv.FormatUint(uint64(req.MessageID), 10))},
	}
	return &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &kafkaTopic, Partition: kafka.PartitionAny},
		Key:            []byte(req.TopicName),
		Value:          req.Message,
		Opaque:         opaque,
		Headers:        headers,
	}, nil
}

func (s *Publisher) Publish(ctx context.Context, request *apis.PublishRequest) (*apis.PublishResponse, error) {
	producer := s.producers[request.Qos]
	if producer == nil {
		return nil, errors.Errorf("kafka producer for qos %d not found", request.Qos)
	}

	msg, err := s.newKafkaMessage(request, nil)
	if err != nil {
		return nil, err
	}

	deliveryChan := make(chan kafka.Event, 1)
	err = producer.Produce(msg, deliveryChan)
	if err != nil {
		return nil, err
	}
	select {
	case event := <-deliveryChan:
		switch e := event.(type) {
		case *kafka.Message:
			return &apis.PublishResponse{ID: &e.TopicPartition, Error: e.TopicPartition.Error}, nil
		default:
			return nil, errors.Errorf("unexpected event type: %v: %v", reflect.TypeOf(e), e)
		}
	case <-ctx.Done():
		return nil, errors.New("context done")
	}
}

type publishCallback struct {
	request  *apis.PublishRequest
	callback apis.PublishCallbackFunc
}

func (s *Publisher) PublishAsync(_ context.Context, request *apis.PublishRequest, callback apis.PublishCallbackFunc) error {

	producer := s.producers[request.Qos]
	if producer == nil {
		return errors.Errorf("kafka producer for qos %d not found", request.Qos)
	}
	msg, err := s.newKafkaMessage(request, &publishCallback{request: request, callback: callback})
	if err != nil {
		return err
	}
	err = producer.Produce(msg, nil)
	if err != nil {
		return err
	}
	return nil
}

func (s *Publisher) Serve() error {
	defer s.workersDone.Close()

	workers := s.opts.workers
	if workers < 1 {
		workers = 1
	}
	for _, producer := range s.producers {
		for worker := 0; worker < workers; worker++ {
			go func(p *kafkaProducer, worker int) {
				defer s.workersDone.Close()
				logger := p.logger.WithField("worker", strconv.Itoa(worker))
				defer logger.Infof("Terminate worker")
				deliveryReportEventLoop(s.workersDone, logger, p.Events())
			}(producer, worker)
		}
	}

	select {
	case <-s.workersDone.Done():
		s.logger.Infof("received workers done signal")
	}
	return nil
}

func deliveryReportEventLoop(doneChannel *runtime.DoneChannel, logger log.Logger, events chan kafka.Event) {
	for {
		select {
		case e := <-events:
			switch ev := e.(type) {
			case *kafka.Message:
				opaque, ok := ev.Opaque.(*publishCallback)
				if ok {
					opaque.callback(opaque.request, &apis.PublishResponse{ID: &ev.TopicPartition, Error: ev.TopicPartition.Error})
				} else {
					logger.Errorf("unexpected opaque type %v: %v", reflect.TypeOf(opaque), ev)
				}
			case kafka.Error:
				ke := ev
				if ke.IsFatal() {
					logger.WithError(ke).Errorf("fatal kafka error, exiting delivery loop")
					return
				} else {
					logger.WithError(ke).Errorf("kafka error")
				}
			default:
				if e == nil {
					// assume channel was closed
					logger.Infof("null event received, exiting delivery loop")
					return
				}
				logger.Debugf("ignored event type: %v: %v", reflect.TypeOf(e), ev)
			}
		case <-doneChannel.Done():
			logger.Infof("done received, exiting delivery loop")
			return
		}
	}
}
