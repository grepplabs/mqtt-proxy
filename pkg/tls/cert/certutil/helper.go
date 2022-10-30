package certutil

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

const (
	X509CRLBlockType     = "X509 CRL"
	CertificateBlockType = "CERTIFICATE"
)

func ParseCRLsPEM(pemCrls []byte) ([]*x509.RevocationList, error) {
	ok := false
	var lists []*x509.RevocationList
	for len(pemCrls) > 0 {
		var block *pem.Block
		block, pemCrls = pem.Decode(pemCrls)
		if block == nil {
			break
		}
		if block.Type != X509CRLBlockType {
			continue
		}
		list, err := x509.ParseRevocationList(block.Bytes)
		if err != nil {
			return lists, err
		}
		lists = append(lists, list)
		ok = true
	}
	if !ok {
		return lists, errors.New("data does not contain any valid CRL")
	}
	return lists, nil
}

func ParseCertsPEM(pemCerts []byte) ([]*x509.Certificate, error) {
	ok := false
	var certs []*x509.Certificate
	for len(pemCerts) > 0 {
		var block *pem.Block
		block, pemCerts = pem.Decode(pemCerts)
		if block == nil {
			break
		}
		if block.Type != CertificateBlockType || len(block.Headers) != 0 {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return certs, err
		}

		certs = append(certs, cert)
		ok = true
	}

	if !ok {
		return certs, errors.New("data does not contain any valid RSA or ECDSA certificates")
	}
	return certs, nil
}

func GetHexFormatted(buf []byte, sep string) string {
	var ret bytes.Buffer
	for _, cur := range buf {
		if ret.Len() > 0 {
			_, _ = fmt.Fprintf(&ret, sep)
		}
		_, _ = fmt.Fprintf(&ret, "%02x", cur)
	}
	return ret.String()
}
