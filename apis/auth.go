package apis

import "context"

type UserPasswordAuthRequest struct {
	Username string
	Password string
}

type UserPasswordAuthResponse struct {
	ReturnCode byte
}

type UserPasswordAuthenticator interface {
	Login(context.Context, *UserPasswordAuthRequest) (*UserPasswordAuthResponse, error)
	Close() error
}

type UserPasswordAuthenticatorFactory interface {
	New(context.Context) (UserPasswordAuthenticator, error)
}
