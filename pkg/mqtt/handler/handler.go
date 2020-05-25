package mqtthandler

import (
	"context"
	"github.com/pkg/errors"
	"time"

	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	mqttcodec "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec"
	mqttserver "github.com/grepplabs/mqtt-proxy/pkg/mqtt/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type MQTTHandler struct {
	mux       *mqttserver.ServeMux
	logger    log.Logger
	metrics   *mqttMetrics
	publisher apis.Publisher

	opts options
}

type mqttMetrics struct {
	requestsTotal  *prometheus.CounterVec
	responsesTotal *prometheus.CounterVec
}

func (h *MQTTHandler) ServeMQTT(c mqttserver.Conn, p mqttcodec.ControlPacket) {
	h.metrics.requestsTotal.WithLabelValues(p.Name()).Inc()
	h.mux.ServeMQTT(c, p)
}

func (h *MQTTHandler) HandleFunc(messageType byte, handlerFunc mqttserver.HandlerFunc) {
	h.mux.Handle(messageType, handlerFunc)
}

func (h *MQTTHandler) disconnectUnauthenticated(conn mqttserver.Conn, packet mqttcodec.ControlPacket) bool {
	if conn.Properties().Authenticated() {
		return false
	}
	name := packet.Name()
	for _, v := range h.opts.allowUnauthenticated {
		if v == name {
			return false
		}
	}
	h.logger.Warnf("Unauthenticated '%s' from /%v", name, conn.RemoteAddr())
	_ = conn.Close()
	return true
}

func (h *MQTTHandler) handleConnect(conn mqttserver.Conn, packet mqttcodec.ControlPacket) {
	req := packet.(*mqttcodec.ConnectPacket)

	h.logger.Infof("Handling MQTT message '%s' from /%v", req.Name(), conn.RemoteAddr())

	//TODO: login with user password

	if req.KeepAliveSeconds > 0 {
		conn.Properties().SetIdleTimeout(time.Duration(float64(req.KeepAliveSeconds)*1.5) * time.Second)
	}
	conn.Properties().SetAuthenticated(true)

	res := mqttcodec.NewControlPacket(mqttcodec.CONNACK)
	err := res.Write(conn)
	if err != nil {
		h.logger.WithError(err).Errorf("Write 'CONNACK' failed")
	} else {
		h.metrics.responsesTotal.WithLabelValues(res.Name()).Inc()
	}
}

func (h *MQTTHandler) handlePublish(conn mqttserver.Conn, packet mqttcodec.ControlPacket) {
	req := packet.(*mqttcodec.PublishPacket)

	if h.disconnectUnauthenticated(conn, packet) {
		return
	}

	h.logger.Debugf("Handling MQTT message '%s' from /%v", req.Name(), conn.RemoteAddr())

	var publishCallback apis.PublishCallbackFunc

	switch req.Qos {
	case mqttcodec.AT_MOST_ONCE:
		publishCallback = func(*apis.PublishRequest, *apis.PublishResponse) {
			// nothing to send back, publishCallback can be used for metrics
		}
	case mqttcodec.AT_LEAST_ONCE:
		publishCallback = func(request *apis.PublishRequest, response *apis.PublishResponse) {
			if response.Error != nil {
				//TODO: property if close connection unable to deliver ?
				return
			}
			res := mqttcodec.NewControlPacket(mqttcodec.PUBACK).(*mqttcodec.PubackPacket)
			res.MessageID = request.MessageID
			err := res.Write(conn)
			if err != nil {
				h.logger.WithError(err).Errorf("Write 'PUBACK' failed")
			} else {
				h.metrics.responsesTotal.WithLabelValues(res.Name()).Inc()
			}
		}
	case mqttcodec.EXACTLY_ONCE:
		publishCallback = func(req *apis.PublishRequest, resp *apis.PublishResponse) {
			if resp.Error != nil {
				//TODO: property if close connection unable to deliver ?
				return
			}
			res := mqttcodec.NewControlPacket(mqttcodec.PUBREC).(*mqttcodec.PubrecPacket)
			res.MessageID = req.MessageID
			err := res.Write(conn)
			if err != nil {
				h.logger.WithError(err).Errorf("Write 'PUBREC' failed")
			} else {
				h.metrics.responsesTotal.WithLabelValues(res.Name()).Inc()
			}
		}
	default:
		h.logger.Warnf("'PUBLISH' with invalid QoS '%d'. Ignoring", req.Qos)
		return
	}

	ctx := context.Background()
	if h.opts.publishTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.opts.publishTimeout)
		defer cancel()
	}
	err := h.doPublish(ctx, h.publisher, req, publishCallback)
	if err != nil {
		if req.Qos == mqttcodec.AT_MOST_ONCE {
			h.logger.WithError(err).Warnf("Write 'PUBLISH' failed, ignoring ...")
		} else {
			h.logger.WithError(err).Errorf("Write 'PUBLISH' failed, closing the connection ...")
			_ = conn.Close()
		}
	}
}

func (h *MQTTHandler) doPublish(ctx context.Context, publisher apis.Publisher, req *mqttcodec.PublishPacket, publishCallback apis.PublishCallbackFunc) error {
	publishRequest := newPublishRequest(req)
	if h.isPublishAsync(publishRequest.Qos) {
		err := publisher.PublishAsync(ctx, publishRequest, publishCallback)
		if err != nil {
			return errors.Wrap(err, "async publish failed")
		}
	} else {
		publishResponse, err := publisher.Publish(ctx, publishRequest)
		if err != nil {
			return errors.Wrap(err, "sync publish failed")
		}
		publishCallback(publishRequest, publishResponse)
	}
	return nil
}

func (h MQTTHandler) isPublishAsync(qos byte) bool {
	switch qos {
	case mqttcodec.AT_MOST_ONCE:
		return h.opts.publishAsyncAtMostOnce
	case mqttcodec.AT_LEAST_ONCE:
		return h.opts.publishAsyncAtLeastOnce
	case mqttcodec.EXACTLY_ONCE:
		return h.opts.publishAsyncExactlyOnce
	}
	return false
}

func newPublishRequest(req *mqttcodec.PublishPacket) *apis.PublishRequest {
	return &apis.PublishRequest{
		Dup:       req.Dup,
		Qos:       req.Qos,
		Retain:    req.Retain,
		TopicName: req.TopicName,
		MessageID: req.MessageID,
		Message:   req.Message,
	}
}

func (h *MQTTHandler) handlePublishRelease(conn mqttserver.Conn, packet mqttcodec.ControlPacket) {
	req := packet.(*mqttcodec.PubrelPacket)

	if h.disconnectUnauthenticated(conn, packet) {
		return
	}

	h.logger.Debugf("Handling MQTT message '%s' from /%v", req.Name(), conn.RemoteAddr())
	res := mqttcodec.NewControlPacket(mqttcodec.PUBCOMP).(*mqttcodec.PubcompPacket)
	res.MessageID = req.MessageID
	err := res.Write(conn)
	if err != nil {
		h.logger.WithError(err).Errorf("Write 'PUBCOMP' failed")
	}
}

func (h *MQTTHandler) handlePing(conn mqttserver.Conn, packet mqttcodec.ControlPacket) {
	req := packet.(*mqttcodec.PingreqPacket)

	if h.disconnectUnauthenticated(conn, packet) {
		return
	}

	h.logger.Debugf("Handling MQTT message '%s' from /%v", req.Name(), conn.RemoteAddr())
	res := mqttcodec.NewControlPacket(mqttcodec.PINGRESP)
	err := res.Write(conn)
	if err != nil {
		h.logger.WithError(err).Errorf("Write 'PINGRESP' failed")
	}
}

func (h *MQTTHandler) handleDisconnect(conn mqttserver.Conn, packet mqttcodec.ControlPacket) {
	req := packet.(*mqttcodec.DisconnectPacket)
	h.logger.Infof("Handling MQTT message '%s' from /%v", req.Name(), conn.RemoteAddr())
	err := conn.Close()
	if err != nil {
		h.logger.WithError(err).Warnf("Closing connection on 'DISCONNECT' failed")
	}
}

func (h *MQTTHandler) ignore(conn mqttserver.Conn, packet mqttcodec.ControlPacket) {
	h.logger.Debugf("No handler available for MQTT message '%s' from /%v. Ignoring", packet.Name(), conn.RemoteAddr())
}

func New(logger log.Logger, registry *prometheus.Registry, publisher apis.Publisher, opts ...Option) *MQTTHandler {
	options := options{}
	for _, o := range opts {
		o.apply(&options)
	}
	h := &MQTTHandler{
		mux:       mqttserver.NewServeMux(logger),
		logger:    logger,
		opts:      options,
		metrics:   newMQTTMetrics(registry),
		publisher: publisher,
	}
	h.HandleFunc(mqttcodec.CONNECT, h.handleConnect)
	h.HandleFunc(mqttcodec.PUBLISH, h.handlePublish)
	h.HandleFunc(mqttcodec.DISCONNECT, h.handleDisconnect)
	h.HandleFunc(mqttcodec.PUBREL, h.handlePublishRelease)
	h.HandleFunc(mqttcodec.PINGREQ, h.handlePing)

	for _, name := range options.ignoreUnsupported {
		for t, n := range mqttcodec.MqttMessageTypeNames {
			if n == name {
				logger.Infof("%s requests will be ignored", name)
				h.HandleFunc(t, h.ignore)
				continue
			}
		}
	}
	for _, name := range options.allowUnauthenticated {
		logger.Infof("%s requests will be allow unauthenticated", name)
	}
	return h

}

func newMQTTMetrics(registry *prometheus.Registry) *mqttMetrics {
	requestsTotal := promauto.With(registry).NewCounterVec(prometheus.CounterOpts{
		Name: "mqtt_proxy_handler_requests_total",
		Help: "Total number of MQTT requests.",
	}, []string{"type"})

	requestsTotal.WithLabelValues(mqttcodec.MqttMessageTypeNames[mqttcodec.CONNECT])
	requestsTotal.WithLabelValues(mqttcodec.MqttMessageTypeNames[mqttcodec.PUBLISH])
	requestsTotal.WithLabelValues(mqttcodec.MqttMessageTypeNames[mqttcodec.DISCONNECT])
	requestsTotal.WithLabelValues(mqttcodec.MqttMessageTypeNames[mqttcodec.PUBREL])
	requestsTotal.WithLabelValues(mqttcodec.MqttMessageTypeNames[mqttcodec.PINGREQ])
	requestsTotal.WithLabelValues(mqttcodec.MqttMessageTypeNames[mqttcodec.SUBSCRIBE])
	requestsTotal.WithLabelValues(mqttcodec.MqttMessageTypeNames[mqttcodec.UNSUBSCRIBE])

	responsesTotal := promauto.With(registry).NewCounterVec(prometheus.CounterOpts{
		Name: "mqtt_proxy_handler_responses_total",
		Help: "Total number of MQTT responses.",
	}, []string{"type"})

	responsesTotal.WithLabelValues(mqttcodec.MqttMessageTypeNames[mqttcodec.CONNACK])
	responsesTotal.WithLabelValues(mqttcodec.MqttMessageTypeNames[mqttcodec.PUBACK])
	responsesTotal.WithLabelValues(mqttcodec.MqttMessageTypeNames[mqttcodec.PUBREC])
	responsesTotal.WithLabelValues(mqttcodec.MqttMessageTypeNames[mqttcodec.PUBCOMP])
	responsesTotal.WithLabelValues(mqttcodec.MqttMessageTypeNames[mqttcodec.SUBACK])
	responsesTotal.WithLabelValues(mqttcodec.MqttMessageTypeNames[mqttcodec.UNSUBACK])
	responsesTotal.WithLabelValues(mqttcodec.MqttMessageTypeNames[mqttcodec.PINGRESP])

	return &mqttMetrics{
		requestsTotal:  requestsTotal,
		responsesTotal: responsesTotal,
	}
}
