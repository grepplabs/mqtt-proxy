package tls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/grepplabs/mqtt-proxy/pkg/tls/cert/certutil"
	"github.com/grepplabs/mqtt-proxy/pkg/tls/cert/source"
)

const (
	initLoadTimeout = 15 * time.Second
)

// MustNewServerConfig is like NewServerConfig but panics if the config cannot be created.
func MustNewServerConfig(logger log.Logger, src source.ServerSource) *tls.Config {
	c, err := NewServerConfig(logger, src)
	if err != nil {
		panic(`tls: NewServerConfig(): ` + err.Error())
	}
	return c
}

// NewServerConfig provides new server TLS configuration.
func NewServerConfig(logger log.Logger, src source.ServerSource) (*tls.Config, error) {
	store := source.NewStore(logger)
	logger.Infof("initial server certs loading")

	certsChan := src.ServerCerts()

	select {
	case certs := <-certsChan:
		store.SetServerCerts(certs)
	case <-time.After(initLoadTimeout):
		return nil, errors.New("get server certs timeout")
	}

	go func() {
		for certs := range certsChan {
			store.SetServerCerts(certs)
		}
	}()

	return &tls.Config{
		GetConfigForClient: func(info *tls.ClientHelloInfo) (*tls.Config, error) {
			cs := store.CertStore()
			x := &tls.Config{
				MinVersion:   tls.VersionTLS12,
				Certificates: cs.Certificates,
			}
			if cs.ClientCAs != nil {
				x.ClientCAs = cs.ClientCAs
				x.ClientAuth = tls.RequireAndVerifyClientCert
				x.VerifyPeerCertificate = verifyClientCertificate(logger, store)
			}
			return x, nil
		},
	}, nil
}

func verifyClientCertificate(logger log.Logger, store *source.Store) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		cs := store.CertStore()
		if len(cs.ClientCRLs) == 0 {
			return nil
		}
		for _, chain := range verifiedChains {
			for _, cert := range chain {
				if !cert.IsCA {
					if cs.IsClientCertRevoked(cert.SerialNumber) {
						err := fmt.Errorf("client certificte %s was revoked", certutil.GetHexFormatted(cert.SerialNumber.Bytes(), ":"))
						logger.Debug(err.Error())
						return err
					}
				}
			}
		}
		return nil
	}
}
