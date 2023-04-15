package mqtthandler

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
	mqtt311 "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/v311"
	mqtt5 "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/v5"
	mqttserver "github.com/grepplabs/mqtt-proxy/pkg/mqtt/server"
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

func (h *MQTTHandler) disconnectUnauthenticated(conn mqttserver.Conn, packetName string) bool {
	if conn.Properties().Authenticated() {
		return false
	}
	for _, v := range h.opts.allowUnauthenticated {
		if v == packetName {
			return false
		}
	}
	h.logger.Warnf("Unauthenticated '%s' from /%v", packetName, conn.RemoteAddr())
	_ = conn.Close()
	return true
}

func (h *MQTTHandler) handleConnect(conn mqttserver.Conn, packet mqttproto.ControlPacket) {
	username, password, clientIdentifier, keepAliveSeconds, err := h.getConnectData(packet)
	if err != nil {
		h.logger.Error(err.Error())
		_ = conn.Close()
		return
	}
	h.logger.Infof("Handling MQTT message '%s' from /%v", packet.Name(), conn.RemoteAddr())

	returnCode, err := h.loginUser(username, password)
	if err != nil {
		h.logger.WithError(err).Warnf("Login failed from /%v failed", conn.RemoteAddr())
		_ = conn.Close()
		return
	}
	if keepAliveSeconds > 0 {
		conn.Properties().SetIdleTimeout(time.Duration(float64(keepAliveSeconds)*1.5) * time.Second)
	}
	authenticated := returnCode == mqttproto.Accepted
	conn.Properties().SetAuthenticated(authenticated)
	conn.Properties().SetClientIdentifier(clientIdentifier)

	res, err := h.getConnectAck(packet, returnCode)
	if err != nil {
		h.logger.Error(err.Error())
		_ = conn.Close()
		return
	}
	err = res.Write(conn)
	if err != nil {
		h.logger.WithError(err).Errorf("Write 'CONNACK' failed")
	} else {
		h.metrics.responsesTotal.WithLabelValues(res.Name(), mqttproto.MqttProtocolVersionName(res.Version())).Inc()
	}
	if !authenticated {
		h.logger.Infof("Disconnect unauthenticated user '%s' from /%v", username, conn.RemoteAddr())
		_ = conn.Close()
		return
	}
}

func (h *MQTTHandler) getConnectData(packet mqttproto.ControlPacket) (username string, password string, clientIdentifier string, keepAliveSeconds uint16, err error) {
	switch req := packet.(type) {
	case *mqtt311.ConnectPacket:
		return req.Username, string(req.Password), req.ClientIdentifier, req.KeepAliveSeconds, nil
	case *mqtt5.ConnectPacket:
		return req.Username, string(req.Password), req.ClientIdentifier, req.KeepAliveSeconds, nil
	default:
		return "", "", "", 0, fmt.Errorf("unsupported connect packet type %v", reflect.TypeOf(packet))
	}
}

func (h *MQTTHandler) getConnectAck(packet mqttproto.ControlPacket, returnCode byte) (mqttproto.ControlPacket, error) {
	switch packet.(type) {
	case *mqtt311.ConnectPacket:
		res := mqtt311.NewControlPacket(mqttproto.CONNACK).(*mqtt311.ConnackPacket)
		res.ReturnCode = returnCode
		return res, nil
	case *mqtt5.ConnectPacket:
		res := mqtt5.NewControlPacket(mqttproto.CONNACK).(*mqtt5.ConnackPacket)
		res.ReturnCode = returnCode
		switch returnCode {
		case mqttproto.RefusedBadUserNameOrPassword:
			res.ReturnCode = mqttproto.RefusedV5BadUserNameOrPassword
		}
		return res, nil
	default:
		return nil, fmt.Errorf("unsupported connect packet type %v", reflect.TypeOf(packet))
	}
}

func (h *MQTTHandler) loginUser(username, password string) (byte, error) {
	if h.opts.authenticator != nil {
		authResp, err := h.opts.authenticator.Login(context.Background(), &apis.UserPasswordAuthRequest{
			Username: username,
			Password: password,
		})
		if err != nil {
			return 0, err
		}
		return authResp.ReturnCode, nil
	}
	return mqttproto.Accepted, nil
}

func (h *MQTTHandler) handlePublish(conn mqttserver.Conn, packet mqttproto.ControlPacket) {
	publishRequest, err := h.getPublishRequest(conn, packet)
	if err != nil {
		h.logger.Error(err.Error())
		_ = conn.Close()
		return
	}
	if h.disconnectUnauthenticated(conn, packet.Name()) {
		return
	}
	h.logger.Debugf("Handling MQTT message '%s' from /%v", packet.Name(), conn.RemoteAddr())

	var publishCallback apis.PublishCallbackFunc

	switch publishRequest.Qos {
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
			res, err := h.getPublishAck(packet, request.MessageID)
			if err != nil {
				h.logger.Error(err.Error())
				_ = conn.Close()
				return
			}
			err = res.Write(conn)
			if err != nil {
				h.logger.WithError(err).Errorf("Write 'PUBACK' failed")
			} else {
				h.metrics.responsesTotal.WithLabelValues(res.Name(), mqttproto.MqttProtocolVersionName(res.Version())).Inc()
			}
		}
	case mqttproto.EXACTLY_ONCE:
		publishCallback = func(request *apis.PublishRequest, response *apis.PublishResponse) {
			if response.Error != nil {
				//TODO: property if close connection unable to deliver ?
				return
			}
			res, err := h.getPublishRec(packet, request.MessageID)
			if err != nil {
				h.logger.Error(err.Error())
				_ = conn.Close()
				return
			}
			err = res.Write(conn)
			if err != nil {
				h.logger.WithError(err).Errorf("Write 'PUBREC' failed")
			} else {
				h.metrics.responsesTotal.WithLabelValues(res.Name(), mqttproto.MqttProtocolVersionName(res.Version())).Inc()
			}
		}
	default:
		h.logger.Warnf("'PUBLISH' with invalid QoS '%d'. Ignoring", publishRequest.Qos)
		return
	}

	ctx := context.Background()
	if h.opts.publishTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.opts.publishTimeout)
		defer cancel()
	}
	err = h.doPublish(ctx, h.publisher, publishRequest, publishCallback)
	if err != nil {
		if publishRequest.Qos == mqttproto.AT_MOST_ONCE {
			h.logger.WithError(err).Warnf("Write 'PUBLISH' failed, ignoring ...")
		} else {
			h.logger.WithError(err).Errorf("Write 'PUBLISH' failed, closing the connection ...")
			_ = conn.Close()
		}
	}
}

func (h *MQTTHandler) getPublishAck(packet mqttproto.ControlPacket, messageID uint16) (mqttproto.ControlPacket, error) {
	switch packet.(type) {
	case *mqtt311.PublishPacket:
		res := mqtt311.NewControlPacket(mqttproto.PUBACK).(*mqtt311.PubackPacket)
		res.MessageID = messageID
		return res, nil
	case *mqtt5.PublishPacket:
		res := mqtt5.NewControlPacket(mqttproto.PUBACK).(*mqtt5.PubackPacket)
		res.MessageID = messageID
		res.ReasonCode = 0
		return res, nil
	default:
		return nil, fmt.Errorf("unsupported publish packet type %v", reflect.TypeOf(packet))
	}
}

func (h *MQTTHandler) getPublishRec(packet mqttproto.ControlPacket, messageID uint16) (mqttproto.ControlPacket, error) {
	switch packet.(type) {
	case *mqtt311.PublishPacket:
		res := mqtt311.NewControlPacket(mqttproto.PUBREC).(*mqtt311.PubrecPacket)
		res.MessageID = messageID
		return res, nil
	case *mqtt5.PublishPacket:
		res := mqtt5.NewControlPacket(mqttproto.PUBREC).(*mqtt5.PubrecPacket)
		res.MessageID = messageID
		res.ReasonCode = 0
		return res, nil
	default:
		return nil, fmt.Errorf("unsupported publish packet type %v", reflect.TypeOf(packet))
	}
}

func (h *MQTTHandler) doPublish(ctx context.Context, publisher apis.Publisher, publishRequest *apis.PublishRequest, publishCallback apis.PublishCallbackFunc) error {
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

func (h *MQTTHandler) isPublishAsync(qos byte) bool {
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

func (h *MQTTHandler) getPublishRequest(conn mqttserver.Conn, packet mqttproto.ControlPacket) (*apis.PublishRequest, error) {
	switch req := packet.(type) {
	case *mqtt311.PublishPacket:
		return &apis.PublishRequest{
			Dup:       req.Dup,
			Qos:       req.Qos,
			Retain:    req.Retain,
			TopicName: req.TopicName,
			MessageID: req.MessageID,
			Message:   req.Message,
			ClientID:  conn.Properties().ClientIdentifier(),
		}, nil
	case *mqtt5.PublishPacket:
		return &apis.PublishRequest{
			Dup:       req.Dup,
			Qos:       req.Qos,
			Retain:    req.Retain,
			TopicName: req.TopicName,
			MessageID: req.MessageID,
			Message:   req.Message,
			ClientID:  conn.Properties().ClientIdentifier(),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported publish packet type %v", reflect.TypeOf(packet))
	}
}

func (h *MQTTHandler) handlePublishRelease(conn mqttserver.Conn, packet mqttproto.ControlPacket) {
	if h.disconnectUnauthenticated(conn, packet.Name()) {
		return
	}
	h.logger.Debugf("Handling MQTT message '%s' from /%v", packet.Name(), conn.RemoteAddr())
	res, err := h.getPublishComp(packet)
	if err != nil {
		h.logger.Error(err.Error())
		_ = conn.Close()
		return
	}
	err = res.Write(conn)
	if err != nil {
		h.logger.WithError(err).Errorf("Write 'PUBCOMP' failed")
	}
}

func (h *MQTTHandler) getPublishComp(packet mqttproto.ControlPacket) (mqttproto.ControlPacket, error) {
	switch req := packet.(type) {
	case *mqtt311.PubrelPacket:
		res := mqtt311.NewControlPacket(mqttproto.PUBCOMP).(*mqtt311.PubcompPacket)
		res.MessageID = req.MessageID
		return res, nil
	case *mqtt5.PubrelPacket:
		res := mqtt5.NewControlPacket(mqttproto.PUBCOMP).(*mqtt5.PubcompPacket)
		res.MessageID = req.MessageID
		res.ReasonCode = 0
		return res, nil
	default:
		return nil, fmt.Errorf("unsupported pubrel packet type %v", reflect.TypeOf(packet))
	}
}

func (h *MQTTHandler) handlePing(conn mqttserver.Conn, packet mqttproto.ControlPacket) {
	if h.disconnectUnauthenticated(conn, packet.Name()) {
		return
	}
	h.logger.Debugf("Handling MQTT message '%s' from /%v", packet.Name(), conn.RemoteAddr())
	var res mqttproto.ControlPacket
	switch packet.(type) {
	case *mqtt311.PingreqPacket:
		res = mqtt311.NewControlPacket(mqttproto.PINGRESP)
	case *mqtt5.PingreqPacket:
		res = mqtt5.NewControlPacket(mqttproto.PINGRESP)
	default:
		h.logger.Warnf("Unsupported disconnect ping type %v", reflect.TypeOf(packet))
		return
	}
	err := res.Write(conn)
	if err != nil {
		h.logger.WithError(err).Errorf("Write 'PINGRESP' failed")
	}
}

func (h *MQTTHandler) handleDisconnect(conn mqttserver.Conn, packet mqttproto.ControlPacket) {
	switch packet.(type) {
	case *mqtt311.DisconnectPacket:
	case *mqtt5.DisconnectPacket:
	default:
		h.logger.Warnf("Unsupported disconnect packet type %v", reflect.TypeOf(packet))
	}
	h.logger.Infof("Handling MQTT message '%s' from /%v", packet.Name(), conn.RemoteAddr())
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
