package mqtthandler

import (
	"context"
	"fmt"
	"time"

	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
	mqtt311 "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/v311"
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

func (h *MQTTHandler) ServeMQTT(c mqttserver.Conn, p mqttproto.ControlPacket) {
	h.metrics.requestsTotal.WithLabelValues(p.Name(), mqttproto.MqttProtocolVersionName(p.Version())).Inc()
	h.mux.ServeMQTT(c, p)
}

func (h *MQTTHandler) HandleFunc(messageType byte, handlerFunc mqttserver.HandlerFunc) {
	h.mux.Handle(messageType, handlerFunc)
}

func (h *MQTTHandler) disconnectUnauthenticated(conn mqttserver.Conn, packet mqttproto.ControlPacket) bool {
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

func (h *MQTTHandler) handleConnect(conn mqttserver.Conn, packet mqttproto.ControlPacket) {
	req := packet.(*mqtt311.ConnectPacket)

	h.logger.Infof("Handling MQTT message '%s' from /%v", req.Name(), conn.RemoteAddr())

	returnCode, err := h.loginUser(req)
	if err != nil {
		h.logger.WithError(err).Warnf("Login failed from /%v failed", conn.RemoteAddr())
		_ = conn.Close()
		return
	}
	if req.KeepAliveSeconds > 0 {
		conn.Properties().SetIdleTimeout(time.Duration(float64(req.KeepAliveSeconds)*1.5) * time.Second)
	}
	authenticated := returnCode == mqttproto.Accepted
	conn.Properties().SetAuthenticated(authenticated)

	res := mqtt311.NewControlPacket(mqttproto.CONNACK).(*mqtt311.ConnackPacket)
	res.ReturnCode = returnCode

	err = res.Write(conn)
	if err != nil {
		h.logger.WithError(err).Errorf("Write 'CONNACK' failed")
	} else {
		h.metrics.responsesTotal.WithLabelValues(res.Name(), mqttproto.MqttProtocolVersionName(res.Version())).Inc()
	}
	if !authenticated {
		h.logger.Infof("Disconnect unauthenticated user '%s' from /%v", req.Username, conn.RemoteAddr())
		_ = conn.Close()
		return
	}
}

func (h *MQTTHandler) loginUser(packet *mqtt311.ConnectPacket) (byte, error) {
	if h.opts.authenticator != nil {
		authResp, err := h.opts.authenticator.Login(context.Background(), &apis.UserPasswordAuthRequest{
			Username: packet.Username,
			Password: string(packet.Password),
		})
		if err != nil {
			return 0, err
		}
		return authResp.ReturnCode, nil
	}
	return mqttproto.Accepted, nil
}

func (h *MQTTHandler) handlePublish(conn mqttserver.Conn, packet mqttproto.ControlPacket) {
	req := packet.(*mqtt311.PublishPacket)

	if h.disconnectUnauthenticated(conn, packet) {
		return
	}

	h.logger.Debugf("Handling MQTT message '%s' from /%v", req.Name(), conn.RemoteAddr())

	var publishCallback apis.PublishCallbackFunc

	switch req.Qos {
	case mqttproto.AT_MOST_ONCE:
		publishCallback = func(*apis.PublishRequest, *apis.PublishResponse) {
			// nothing to send back, publishCallback can be used for metrics
		}
	case mqttproto.AT_LEAST_ONCE:
		publishCallback = func(request *apis.PublishRequest, response *apis.PublishResponse) {
			if response.Error != nil {
				//TODO: property if close connection unable to deliver ?
				return
			}
			res := mqtt311.NewControlPacket(mqttproto.PUBACK).(*mqtt311.PubackPacket)
			res.MessageID = request.MessageID
			err := res.Write(conn)
			if err != nil {
				h.logger.WithError(err).Errorf("Write 'PUBACK' failed")
			} else {
				h.metrics.responsesTotal.WithLabelValues(res.Name(), mqttproto.MqttProtocolVersionName(res.Version())).Inc()
			}
		}
	case mqttproto.EXACTLY_ONCE:
		publishCallback = func(req *apis.PublishRequest, resp *apis.PublishResponse) {
			if resp.Error != nil {
				//TODO: property if close connection unable to deliver ?
				return
			}
			res := mqtt311.NewControlPacket(mqttproto.PUBREC).(*mqtt311.PubrecPacket)
			res.MessageID = req.MessageID
			err := res.Write(conn)
			if err != nil {
				h.logger.WithError(err).Errorf("Write 'PUBREC' failed")
			} else {
				h.metrics.responsesTotal.WithLabelValues(res.Name(), mqttproto.MqttProtocolVersionName(res.Version())).Inc()
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
		if req.Qos == mqttproto.AT_MOST_ONCE {
			h.logger.WithError(err).Warnf("Write 'PUBLISH' failed, ignoring ...")
		} else {
			h.logger.WithError(err).Errorf("Write 'PUBLISH' failed, closing the connection ...")
			_ = conn.Close()
		}
	}
}

func (h *MQTTHandler) doPublish(ctx context.Context, publisher apis.Publisher, req *mqtt311.PublishPacket, publishCallback apis.PublishCallbackFunc) error {
	publishRequest := newPublishRequest(req)
	if h.isPublishAsync(publishRequest.Qos) {
		err := publisher.PublishAsync(ctx, publishRequest, publishCallback)
		if err != nil {
			return fmt.Errorf("async publish failed: %w", err)
		}
	} else {
		publishResponse, err := publisher.Publish(ctx, publishRequest)
		if err != nil {
			return fmt.Errorf("sync publish failed: %w", err)
		}
		publishCallback(publishRequest, publishResponse)
	}
	return nil
}

func (h MQTTHandler) isPublishAsync(qos byte) bool {
	switch qos {
	case mqttproto.AT_MOST_ONCE:
		return h.opts.publishAsyncAtMostOnce
	case mqttproto.AT_LEAST_ONCE:
		return h.opts.publishAsyncAtLeastOnce
	case mqttproto.EXACTLY_ONCE:
		return h.opts.publishAsyncExactlyOnce
	}
	return false
}

func newPublishRequest(req *mqtt311.PublishPacket) *apis.PublishRequest {
	return &apis.PublishRequest{
		Dup:       req.Dup,
		Qos:       req.Qos,
		Retain:    req.Retain,
		TopicName: req.TopicName,
		MessageID: req.MessageID,
		Message:   req.Message,
	}
}

func (h *MQTTHandler) handlePublishRelease(conn mqttserver.Conn, packet mqttproto.ControlPacket) {
	req := packet.(*mqtt311.PubrelPacket)

	if h.disconnectUnauthenticated(conn, packet) {
		return
	}

	h.logger.Debugf("Handling MQTT message '%s' from /%v", req.Name(), conn.RemoteAddr())
	res := mqtt311.NewControlPacket(mqttproto.PUBCOMP).(*mqtt311.PubcompPacket)
	res.MessageID = req.MessageID
	err := res.Write(conn)
	if err != nil {
		h.logger.WithError(err).Errorf("Write 'PUBCOMP' failed")
	}
}

func (h *MQTTHandler) handlePing(conn mqttserver.Conn, packet mqttproto.ControlPacket) {
	req := packet.(*mqtt311.PingreqPacket)

	if h.disconnectUnauthenticated(conn, packet) {
		return
	}

	h.logger.Debugf("Handling MQTT message '%s' from /%v", req.Name(), conn.RemoteAddr())
	res := mqtt311.NewControlPacket(mqttproto.PINGRESP)
	err := res.Write(conn)
	if err != nil {
		h.logger.WithError(err).Errorf("Write 'PINGRESP' failed")
	}
}

func (h *MQTTHandler) handleDisconnect(conn mqttserver.Conn, packet mqttproto.ControlPacket) {
	req := packet.(*mqtt311.DisconnectPacket)
	h.logger.Infof("Handling MQTT message '%s' from /%v", req.Name(), conn.RemoteAddr())
	err := conn.Close()
	if err != nil {
		h.logger.WithError(err).Warnf("Closing connection on 'DISCONNECT' failed")
	}
}

func (h *MQTTHandler) ignore(conn mqttserver.Conn, packet mqttproto.ControlPacket) {
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
	h.HandleFunc(mqttproto.CONNECT, h.handleConnect)
	h.HandleFunc(mqttproto.PUBLISH, h.handlePublish)
	h.HandleFunc(mqttproto.DISCONNECT, h.handleDisconnect)
	h.HandleFunc(mqttproto.PUBREL, h.handlePublishRelease)
	h.HandleFunc(mqttproto.PINGREQ, h.handlePing)

	for _, name := range options.ignoreUnsupported {
		for t, n := range mqttproto.MqttMessageTypeNames {
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
	}, []string{"type", "version"})

	requestsTotal.WithLabelValues(mqttproto.MqttMessageTypeNames[mqttproto.CONNECT], mqttproto.MqttProtocolVersionName(mqttproto.MQTT_DEFAULT_PROTOCOL_VERSION))
	requestsTotal.WithLabelValues(mqttproto.MqttMessageTypeNames[mqttproto.PUBLISH], mqttproto.MqttProtocolVersionName(mqttproto.MQTT_DEFAULT_PROTOCOL_VERSION))
	requestsTotal.WithLabelValues(mqttproto.MqttMessageTypeNames[mqttproto.DISCONNECT], mqttproto.MqttProtocolVersionName(mqttproto.MQTT_DEFAULT_PROTOCOL_VERSION))
	requestsTotal.WithLabelValues(mqttproto.MqttMessageTypeNames[mqttproto.PUBREL], mqttproto.MqttProtocolVersionName(mqttproto.MQTT_DEFAULT_PROTOCOL_VERSION))
	requestsTotal.WithLabelValues(mqttproto.MqttMessageTypeNames[mqttproto.PINGREQ], mqttproto.MqttProtocolVersionName(mqttproto.MQTT_DEFAULT_PROTOCOL_VERSION))
	requestsTotal.WithLabelValues(mqttproto.MqttMessageTypeNames[mqttproto.SUBSCRIBE], mqttproto.MqttProtocolVersionName(mqttproto.MQTT_DEFAULT_PROTOCOL_VERSION))
	requestsTotal.WithLabelValues(mqttproto.MqttMessageTypeNames[mqttproto.UNSUBSCRIBE], mqttproto.MqttProtocolVersionName(mqttproto.MQTT_DEFAULT_PROTOCOL_VERSION))

	responsesTotal := promauto.With(registry).NewCounterVec(prometheus.CounterOpts{
		Name: "mqtt_proxy_handler_responses_total",
		Help: "Total number of MQTT responses.",
	}, []string{"type", "version"})

	responsesTotal.WithLabelValues(mqttproto.MqttMessageTypeNames[mqttproto.CONNACK], mqttproto.MqttProtocolVersionName(mqttproto.MQTT_DEFAULT_PROTOCOL_VERSION))
	responsesTotal.WithLabelValues(mqttproto.MqttMessageTypeNames[mqttproto.PUBACK], mqttproto.MqttProtocolVersionName(mqttproto.MQTT_DEFAULT_PROTOCOL_VERSION))
	responsesTotal.WithLabelValues(mqttproto.MqttMessageTypeNames[mqttproto.PUBREC], mqttproto.MqttProtocolVersionName(mqttproto.MQTT_DEFAULT_PROTOCOL_VERSION))
	responsesTotal.WithLabelValues(mqttproto.MqttMessageTypeNames[mqttproto.PUBCOMP], mqttproto.MqttProtocolVersionName(mqttproto.MQTT_DEFAULT_PROTOCOL_VERSION))
	responsesTotal.WithLabelValues(mqttproto.MqttMessageTypeNames[mqttproto.SUBACK], mqttproto.MqttProtocolVersionName(mqttproto.MQTT_DEFAULT_PROTOCOL_VERSION))
	responsesTotal.WithLabelValues(mqttproto.MqttMessageTypeNames[mqttproto.UNSUBACK], mqttproto.MqttProtocolVersionName(mqttproto.MQTT_DEFAULT_PROTOCOL_VERSION))
	responsesTotal.WithLabelValues(mqttproto.MqttMessageTypeNames[mqttproto.PINGRESP], mqttproto.MqttProtocolVersionName(mqttproto.MQTT_DEFAULT_PROTOCOL_VERSION))

	return &mqttMetrics{
		requestsTotal:  requestsTotal,
		responsesTotal: responsesTotal,
	}
}
