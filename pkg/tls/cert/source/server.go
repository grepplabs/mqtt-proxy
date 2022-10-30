package source

import (
	"crypto/tls"
	"crypto/x509"
)

type ServerCerts struct {
	Certificates []tls.Certificate
	ClientCAs    *x509.CertPool
	ClientCRLs   []*x509.RevocationList
	Checksum     []byte
}

type ServerSource interface {
	ServerCerts() chan ServerCerts
}
