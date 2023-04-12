package apis

import "context"

// PublishID is optional identifier for a particular message assigned by broker
// It can be complete in case of fire and forget delivery
type PublishID interface{}

type PublishRequest struct {
	Dup       bool   `json:"dup"`
	Qos       byte   `json:"qos"`
	Retain    bool   `json:"retain"`
	TopicName string `json:"topic_name"`
	MessageID uint16 `json:"packet_id"`
	Message   []byte `json:"payload"`
}

type PublishResponse struct {
	ID    PublishID
	Error error
}

type PublishCallbackFunc func(*PublishRequest, *PublishResponse)

type Publisher interface {
	Name() string
	Publish(context.Context, *PublishRequest) (*PublishResponse, error)
	PublishAsync(context.Context, *PublishRequest, PublishCallbackFunc) error
	Serve() error
	Shutdown(err error)
	Close() error
}

type PublisherFactory interface {
	New(params []string) (Publisher, error)
}
