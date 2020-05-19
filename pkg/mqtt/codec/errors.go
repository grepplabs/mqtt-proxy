package mqttcodec

import "fmt"

func NewConnAckError(returnCode byte, text string) error {
	return &ConnectAckError{rc: returnCode, s: text}
}

type ConnectAckError struct {
	rc byte
	s  string
}

func (e *ConnectAckError) Error() string {
	return fmt.Sprintf("CONNACK return code %d: %s", e.rc, e.s)
}

func (e *ConnectAckError) ReturnCode() byte {
	return e.rc
}
