// SPDX-License-Identifier: MIT

package broker

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/otfabric/go-sparkplugb/mqtt"
)

type Opts struct {
	BrokerURL string
	ClientID  string
	Username  string
	Password  string
	TLSCert   string
	TLSKey    string
	TLSCA     string
}

func NewTransport(o Opts) (*mqtt.PahoTransport, error) {
	t := mqtt.NewPahoTransport()

	if o.TLSCert != "" && o.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(o.TLSCert, o.TLSKey)
		if err != nil {
			return nil, fmt.Errorf("load TLS keypair: %w", err)
		}
		tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}

		if o.TLSCA != "" {
			caCert, err := os.ReadFile(o.TLSCA)
			if err != nil {
				return nil, fmt.Errorf("read CA cert: %w", err)
			}
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(caCert)
			tlsCfg.RootCAs = pool
		}
		t.TLSConfig = tlsCfg
	}

	return t, nil
}
