package mqtthandler

import (
	"context"
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

func (h *MQTTHandler) isAuthenticated(conn mqttserver.Conn, messageName string) bool {
	if h.opts.allowUnauthenticated {
		return true
	}
	if !conn.Properties().Authenticated() {
		h.logger.Warnf("Unauthenticated '%s' from /%v", messageName, conn.RemoteAddr())
		_ = conn.Close()
		return false
	}
	return true
}

func handleConnect(h *MQTTHandler) mqttserver.HandlerFunc {
	return func(conn mqttserver.Conn, packet mqttcodec.ControlPacket) {
		req := packet.(*mqttcodec.ConnectPacket)

		h.logger.Infof("Handling MQTT message '%s' from /%v", req.MessageName(), conn.RemoteAddr())

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
}

func handlePublish(h *MQTTHandler) mqttserver.HandlerFunc {
	return func(conn mqttserver.Conn, packet mqttcodec.ControlPacket) {
		req := packet.(*mqttcodec.PublishPacket)

		h.logger.Debugf("Handling MQTT message '%s' from /%v", req.MessageName(), conn.RemoteAddr())

		if !h.isAuthenticated(conn, req.MessageName()) {
			return
		}

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

		publishRequest := newPublishRequest(req)
		// TODO: configure in the properties (sync default ? for all to keep order in kafka)
		// TODO: mosquito cannot handle out order confirmations
		if publishRequest.Qos == 0 {
			err := h.publisher.PublishAsync(context.Background(), publishRequest, publishCallback)
			if err != nil {
				h.logger.WithError(err).Errorf("Write 'PUBLISH' failed")
				//TODO: property if close connection unable to deliver ?
			}
		} else {
			// TODO: can add timeout for publish ?
			publishResponse, err := h.publisher.Publish(context.Background(), publishRequest)
			if err != nil {
				h.logger.WithError(err).Errorf("Write 'PUBLISH' failed")
				//TODO: property if close connection unable to deliver ?
				return
			}
			publishCallback(publishRequest, publishResponse)
		}
	}
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

func handlePublishRelease(h *MQTTHandler) mqttserver.HandlerFunc {
	return func(conn mqttserver.Conn, packet mqttcodec.ControlPacket) {
		req := packet.(*mqttcodec.PubrelPacket)

		if !h.isAuthenticated(conn, req.MessageName()) {
			return
		}

		h.logger.Debugf("Handling MQTT message '%s' from /%v", req.MessageName(), conn.RemoteAddr())
		res := mqttcodec.NewControlPacket(mqttcodec.PUBCOMP).(*mqttcodec.PubcompPacket)
		res.MessageID = req.MessageID
		err := res.Write(conn)
		if err != nil {
			h.logger.WithError(err).Errorf("Write 'PUBCOMP' failed")
		}
	}
}

func handlePing(h *MQTTHandler) mqttserver.HandlerFunc {
	return func(conn mqttserver.Conn, packet mqttcodec.ControlPacket) {
		req := packet.(*mqttcodec.PingreqPacket)

		h.logger.Debugf("Handling MQTT message '%s' from /%v", req.MessageName(), conn.RemoteAddr())
		res := mqttcodec.NewControlPacket(mqttcodec.PINGRESP)
		err := res.Write(conn)
		if err != nil {
			h.logger.WithError(err).Errorf("Write 'PINGRESP' failed")
		}
	}
}

func handleDisconnect(h *MQTTHandler) mqttserver.HandlerFunc {
	return func(conn mqttserver.Conn, packet mqttcodec.ControlPacket) {
		req := packet.(*mqttcodec.DisconnectPacket)
		h.logger.Infof("Handling MQTT message '%s' from /%v", req.MessageName(), conn.RemoteAddr())
		err := conn.Close()
		if err != nil {
			h.logger.WithError(err).Warnf("Closing connection on 'DISCONNECT' failed")
		}
	}
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
	h.mux.Handle(mqttcodec.CONNECT, handleConnect(h))
	h.mux.Handle(mqttcodec.PUBLISH, handlePublish(h))
	h.mux.Handle(mqttcodec.DISCONNECT, handleDisconnect(h))
	h.mux.Handle(mqttcodec.PUBREL, handlePublishRelease(h))
	h.mux.Handle(mqttcodec.PINGREQ, handlePing(h))
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
