package source

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"github.com/grepplabs/mqtt-proxy/pkg/tls/cert/certutil"
)

type CertStore struct {
	Certificates []tls.Certificate
	ClientCAs    *x509.CertPool
	ClientCRLs   []*x509.RevocationList

	buildOnce                sync.Once
	lazyRevokedSerialNumbers map[string]struct{}
}

func (c *CertStore) IsClientCertRevoked(serialNumber *big.Int) bool {
	c.buildOnce.Do(func() {
		c.lazyRevokedSerialNumbers = make(map[string]struct{})
		for _, clientCRL := range c.ClientCRLs {
			for _, revoked := range clientCRL.RevokedCertificates {
				c.lazyRevokedSerialNumbers[string(revoked.SerialNumber.Bytes())] = struct{}{}
			}
		}

	})
	_, ok := c.lazyRevokedSerialNumbers[string(serialNumber.Bytes())]
	return ok
}

type Store struct {
	cs     atomic.Pointer[CertStore]
	logger log.Logger
}

func NewStore(logger log.Logger) *Store {
	s := &Store{
		logger: logger,
	}
	s.cs.Store(&CertStore{})
	return s
}

func (s *Store) CertStore() CertStore {
	return *s.cs.Load()
}

func (s *Store) SetServerCerts(certs ServerCerts) {
	cs := &CertStore{Certificates: certs.Certificates, ClientCAs: certs.ClientCAs, ClientCRLs: certs.ClientCRLs}
	s.cs.Store(cs)
	s.logger.Infof("stored x509 certs for names [%s], clrs %d [%s]", strings.Join(s.names(cs.Certificates), "|"), len(cs.ClientCRLs), strings.Join(s.clrs(cs.ClientCRLs), "|"))
}

func (s *Store) names(certs []tls.Certificate) []string {
	var result []string
	for _, c := range certs {
		x509Cert, err := x509.ParseCertificate(c.Certificate[0])
		if err != nil {
			continue
		}
		var names []string
		if len(x509Cert.Subject.CommonName) > 0 {
			names = append(names, x509Cert.Subject.CommonName)
		}
		for _, san := range x509Cert.DNSNames {
			names = append(names, san)
		}
		result = append(result, fmt.Sprintf("%s=%s", certutil.GetHexFormatted(x509Cert.SerialNumber.Bytes(), ":"), strings.Join(names, ",")))
	}
	return result
}

func (s *Store) clrs(lists []*x509.RevocationList) []string {
	var result []string
	for _, list := range lists {
		for _, cert := range list.RevokedCertificates {
			result = append(result, certutil.GetHexFormatted(cert.SerialNumber.Bytes(), ":"))
		}
	}
	return result
}
