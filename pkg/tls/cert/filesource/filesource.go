package filesource

import (
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/grepplabs/mqtt-proxy/pkg/log"
	tlscert "github.com/grepplabs/mqtt-proxy/pkg/tls/cert/source"
)

const (
	defaultCertFile = "mqtt-proxy-crt.pem"
	defaultKeyFile  = "mqtt-proxy-key.pem"
)

type fileSource struct {
	certFile       string
	keyFile        string
	clientAuthFile string
	clientCRLFile  string
	refresh        time.Duration
	logger         log.Logger

	lastServerCerts atomic.Pointer[tlscert.ServerCerts]
}

func New(opts ...Option) (tlscert.ServerSource, error) {
	s := &fileSource{
		logger: log.NewDefaultLogger(),
	}
	if dir, err := os.Getwd(); err == nil {
		s.certFile = filepath.Join(dir, defaultCertFile)
		s.keyFile = filepath.Join(dir, defaultKeyFile)
	} else {
		return nil, err
	}
	for _, opt := range opts {
		opt(s)
	}
	lastServerCerts, err := s.getServerCerts()
	if err != nil {
		return nil, err
	}
	s.lastServerCerts.Store(lastServerCerts)
	return s, nil
}

func (s *fileSource) getServerCerts() (*tlscert.ServerCerts, error) {
	pemBlocks, err := s.Load()
	if err != nil {
		return nil, err
	}
	certificates, err := pemBlocks.Certificates()
	if err != nil {
		return nil, err
	}
	clientCAs, err := pemBlocks.ClientCAs()
	if err != nil {
		return nil, err
	}
	clientCRLs, err := pemBlocks.ClientCRLs()
	if err != nil {
		return nil, err
	}
	if err = pemBlocks.ValidateCRLs(); err != nil {
		return nil, err
	}
	return &tlscert.ServerCerts{
		Certificates: certificates,
		ClientCAs:    clientCAs,
		ClientCRLs:   clientCRLs,
		Checksum:     pemBlocks.Checksum(),
	}, nil
}

func (s *fileSource) refreshServerCerts() (*tlscert.ServerCerts, error) {
	serverCerts, err := s.getServerCerts()
	if err != nil {
		return nil, err
	}
	s.lastServerCerts.Store(serverCerts)
	return serverCerts, nil
}

func (s *fileSource) ServerCerts() chan tlscert.ServerCerts {
	initialServerCert := s.lastServerCerts.Load()
	ch := make(chan tlscert.ServerCerts, 1)
	if initialServerCert != nil {
		ch <- *initialServerCert
	}
	go func() {
		tlscert.Watch(s.logger, ch, s.refresh, initialServerCert, s.refreshServerCerts)
		close(ch)
	}()
	return ch
}

func (s *fileSource) Load() (pemBlocks *tlscert.ServerPEMs, err error) {
	if s.certFile == "" {
		return nil, errors.New("cert file source: certFile is required")
	}
	if s.keyFile == "" {
		return nil, errors.New("cert file source: keyFile is required")
	}
	if s.clientAuthFile == "" && s.clientCRLFile != "" {
		return nil, errors.New("cert file source: clientAuthFile is required when clientCRLFile is provided")
	}

	pemBlocks = &tlscert.ServerPEMs{}
	if pemBlocks.CertPEMBlock, err = s.readFile(s.certFile); err != nil {
		return nil, err
	}
	if pemBlocks.KeyPEMBlock, err = s.readFile(s.keyFile); err != nil {
		return nil, err
	}
	if pemBlocks.ClientAuthPEMBlock, err = s.readFile(s.clientAuthFile); err != nil {
		return nil, err
	}
	if pemBlocks.CRLPEMBlock, err = s.readFile(s.clientCRLFile); err != nil {
		return nil, err
	}
	return pemBlocks, nil
}

func (s *fileSource) readFile(name string) ([]byte, error) {
	if name == "" {
		return nil, nil
	}
	return os.ReadFile(name)
}
