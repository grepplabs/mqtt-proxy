package apis

import (
	"context"

	mqttcodec "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec"
)

const (
	AuthAccepted     = mqttcodec.Accepted
	AuthUnauthorized = mqttcodec.RefusedNotAuthorized
)

type UserPasswordAuthRequest struct {
	Username string
	Password string
}

type UserPasswordAuthResponse struct {
	ReturnCode byte
}

type UserPasswordAuthenticator interface {
	Name() string
	Login(context.Context, *UserPasswordAuthRequest) (*UserPasswordAuthResponse, error)
	Close() error
}

type UserPasswordAuthenticatorFactory interface {
	New(params []string) (UserPasswordAuthenticator, error)
}
