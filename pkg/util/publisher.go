package util

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/grepplabs/mqtt-proxy/apis"
	"github.com/grepplabs/mqtt-proxy/pkg/config"
)

func GetMessageBody(messageFormat string, request *apis.PublishRequest) ([]byte, error) {
	switch messageFormat {
	case config.MessageFormatPlain:
		return request.Message, nil
	case config.MessageFormatBase64:
		return []byte(base64.StdEncoding.EncodeToString(request.Message)), nil
	case config.MessageFormatJson:
		v, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}
		return v, nil
	default:
		return nil, fmt.Errorf("unsupported message format '%s'", messageFormat)
	}
}
