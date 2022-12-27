package apis

import (
	"context"

	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
)

const (
	AuthAccepted     = mqttproto.Accepted
	AuthUnauthorized = mqttproto.RefusedNotAuthorized
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
