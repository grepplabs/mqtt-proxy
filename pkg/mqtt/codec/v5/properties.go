package v5

import (
	"bytes"
	"fmt"
	mqttproto "github.com/grepplabs/mqtt-proxy/pkg/mqtt/codec/proto"
	"io"
)

type Properties struct {
	// Data without property length
	RawData []byte
}

func (p *Properties) Unpack(r io.Reader) (err error) {
	totalLength, err := mqttproto.DecodeUvarint(r)
	if err != nil {
		return err
	}
	if totalLength < 0 {
		return fmt.Errorf("negative property length %d", totalLength)
	}
	rawData := make([]byte, totalLength)
	n, err := io.ReadFull(r, rawData)
	if err != nil {
		return err
	}
	if n != totalLength {
		return fmt.Errorf("failed to read encoded data, read %d from %d", n, totalLength)
	}
	p.RawData = rawData
	return err
}

func (p *Properties) Write(w io.Writer) (err error) {
	_, err = w.Write(p.Encode())
	return err
}

func (p *Properties) Encode() []byte {
	var buf bytes.Buffer
	mqttproto.WriteUvarint(&buf, uint32(len(p.RawData)))
	buf.Write(p.RawData)
	return buf.Bytes()
}
