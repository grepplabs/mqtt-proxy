package proto

import (
	"io"
)

type ControlPacket interface {
	Write(io.Writer) error
	Unpack(io.Reader) error
	String() string
	Type() byte
	Name() string
	Version() byte
}

type ResponsePacket interface {
	Response() ControlPacket
}
