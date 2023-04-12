package util

import (
	"errors"
	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/grepplabs/mqtt-proxy/pkg/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetMessageBody(t *testing.T) {
	request := &apis.PublishRequest{
		Dup:       false,
		Qos:       1,
		Retain:    true,
		TopicName: "test-topic",
		MessageID: 4711,
		Message:   []byte("hot"),
	}

	tests := []struct {
		name   string
		format string
		body   string
		err    error
	}{
		{
			name:   "plan",
			format: config.MessageFormatPlain,
			body:   "hot",
		},
		{
			name:   "base64",
			format: config.MessageFormatBase64,
			body:   "aG90",
		},
		{
			name:   "json",
			format: config.MessageFormatJson,
			body:   `{"dup":false,"qos":1,"retain":true,"topic_name":"test-topic","packet_id":4711,"payload":"aG90"}`,
		},
		{
			name:   "unsupported",
			format: "na",
			err:    errors.New("unsupported message format 'na'"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, err := GetMessageBody(tc.format, request)
			if err != nil && tc.err != nil {
				require.Equal(t, tc.err, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, tc.body, string(body))
			}
		})
	}
}
