package filesource

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	servertls "github.com/grepplabs/mqtt-proxy/pkg/tls"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestServerConfig(t *testing.T) {
	logger := log.GetInstance()
	bundle := newCertsBundle()
	defer bundle.Close()

	tests := []struct {
		name          string
		transportFunc func() http.RoundTripper
		configFunc    func() *tls.Config
		requestError  bool
	}{
		{
			name: "Client unknown authority",
			transportFunc: func() http.RoundTripper {
				return newRoundTripper()
			},
			configFunc: func() *tls.Config {
				return servertls.MustNewServerConfig(log.GetInstance(), MustNew(
					WithLogger(logger),
					WithX509KeyPair(bundle.ServerCert.Name(), bundle.ServerKey.Name()),
				))
			},
			requestError: true,
		},
		{
			name: "Client insecure",
			transportFunc: func() http.RoundTripper {
				return newRoundTripper(withClientTLSSkipVerify(true))
			},
			configFunc: func() *tls.Config {
				return servertls.MustNewServerConfig(log.GetInstance(), MustNew(
					WithX509KeyPair(bundle.ServerCert.Name(), bundle.ServerKey.Name()),
				))
			},
		},
		{
			name: "Client trusted CA",
			transportFunc: func() http.RoundTripper {
				return newRoundTripper(withRootCAs(bundle.CAX509Cert))
			},
			configFunc: func() *tls.Config {
				return servertls.MustNewServerConfig(log.GetInstance(), MustNew(
					WithX509KeyPair(bundle.ServerCert.Name(), bundle.ServerKey.Name()),
				))
			},
		},
		{
			name: "Client without required certificate",
			transportFunc: func() http.RoundTripper {
				return newRoundTripper(withRootCAs(bundle.CAX509Cert))
			},
			configFunc: func() *tls.Config {
				return servertls.MustNewServerConfig(log.GetInstance(), MustNew(
					WithX509KeyPair(bundle.ServerCert.Name(), bundle.ServerKey.Name()),
					WithClientAuthFile(bundle.CACert.Name()),
				))
			},
			requestError: true,
		},
		{
			name: "Client verification success",
			transportFunc: func() http.RoundTripper {
				return newRoundTripper(withRootCAs(bundle.CAX509Cert), withClientCertificate(bundle.ClientTLSCert))
			},
			configFunc: func() *tls.Config {
				return servertls.MustNewServerConfig(log.GetInstance(), MustNew(
					WithX509KeyPair(bundle.ServerCert.Name(), bundle.ServerKey.Name()),
					WithClientAuthFile(bundle.CACert.Name()),
				))
			},
		},
		{
			name: "Client verification success - empty CRL",
			transportFunc: func() http.RoundTripper {
				return newRoundTripper(withRootCAs(bundle.CAX509Cert), withClientCertificate(bundle.ClientTLSCert))
			},
			configFunc: func() *tls.Config {
				return servertls.MustNewServerConfig(log.GetInstance(), MustNew(
					WithX509KeyPair(bundle.ServerCert.Name(), bundle.ServerKey.Name()),
					WithClientAuthFile(bundle.CACert.Name()),
					WithClientCRLFile(bundle.CAEmptyCRL.Name()),
				))
			},
		},
		{
			name: "Client certificate revoked",
			transportFunc: func() http.RoundTripper {
				return newRoundTripper(withRootCAs(bundle.CAX509Cert), withClientCertificate(bundle.ClientTLSCert))
			},
			configFunc: func() *tls.Config {
				return servertls.MustNewServerConfig(log.GetInstance(), MustNew(
					WithX509KeyPair(bundle.ServerCert.Name(), bundle.ServerKey.Name()),
					WithClientAuthFile(bundle.CACert.Name()),
					WithClientCRLFile(bundle.ClientCRL.Name()),
				))
			},
			requestError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer ts.Close()
			ts.TLS = tc.configFunc()
			ts.StartTLS()

			httpClient := &http.Client{
				Transport: tc.transportFunc(),
			}
			req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
			require.NoError(t, err)

			// when
			res, err := httpClient.Do(req)

			// then
			if tc.requestError {
				t.Log(err)
				require.NotNil(t, err)
				return
			}
			require.NoError(t, err)

			_, err = io.ReadAll(res.Body)
			require.NoError(t, err)

			_ = res.Body.Close()
			require.NoError(t, err)
			require.Equal(t, res.StatusCode, http.StatusOK)

		})
	}
}

func TestCertRotation(t *testing.T) {
	bundle1 := newCertsBundle()
	defer bundle1.Close()

	bundle2 := newCertsBundle()
	defer bundle2.Close()

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	rotatedCh := make(chan struct{}, 1)
	notifyFunc := func() {
		rotatedCh <- struct{}{}
	}
	source := MustNew(
		WithX509KeyPair(bundle1.ServerCert.Name(), bundle1.ServerKey.Name()),
		WithClientAuthFile(bundle1.CACert.Name()),
		WithClientCRLFile(bundle1.CAEmptyCRL.Name()),
		WithRefresh(1*time.Second),
		WithNotifyFunc(notifyFunc),
	).(*fileSource)

	ts.TLS = servertls.MustNewServerConfig(log.GetInstance(), source)
	ts.StartTLS()

	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	require.NoError(t, err)

	// when
	_, err = bundle1.newHttpClient().Do(req)
	require.NoError(t, err)

	// copy new certificates to be used by server
	require.NoError(t, os.Rename(bundle2.ServerCert.Name(), bundle1.ServerCert.Name()))
	require.NoError(t, os.Rename(bundle2.ServerKey.Name(), bundle1.ServerKey.Name()))
	require.NoError(t, os.Rename(bundle2.CACert.Name(), bundle1.CACert.Name()))
	require.NoError(t, os.Rename(bundle2.CAEmptyCRL.Name(), bundle1.CAEmptyCRL.Name()))

	select {
	case <-rotatedCh:
		t.Log("certificates were changed")
		time.Sleep(100 * time.Millisecond)
	case <-time.After(3 * time.Second):
		t.Fatal("expected certificate change notification")
	}
	// old client - bad certificate
	_, err = bundle1.newHttpClient().Do(req)
	require.NotNil(t, err)
	var unknownAuthorityError x509.UnknownAuthorityError
	require.ErrorAs(t, err.(*url.Error).Err, &unknownAuthorityError)

	// new client - success
	_, err = bundle2.newHttpClient().Do(req)
	require.NoError(t, err)
}
