// SPDX-License-Identifier: MIT

package broker

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writePEM(t *testing.T, path, typ string, der []byte) {
	t.Helper()
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
}

func selfSigned(t *testing.T, dir string) (certPath, keyPath string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "sparkplugbctl-test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")
	writePEM(t, certPath, "CERTIFICATE", der)
	writePEM(t, keyPath, "EC PRIVATE KEY", keyDER)
	return certPath, keyPath
}

func TestNewTransportPlain(t *testing.T) {
	tr, err := NewTransport(Opts{BrokerURL: "tcp://localhost:1883"})
	if err != nil {
		t.Fatal(err)
	}
	if tr.TLSConfig != nil {
		t.Fatal("TLS configured without certificates")
	}

	tr, err = NewTransport(Opts{TLSCert: "only-cert.pem"})
	if err != nil || tr.TLSConfig != nil {
		t.Fatalf("cert without key should be ignored: %v %v", tr.TLSConfig, err)
	}
}

func TestNewTransportTLS(t *testing.T) {
	dir := t.TempDir()
	cert, key := selfSigned(t, dir)

	tr, err := NewTransport(Opts{TLSCert: cert, TLSKey: key})
	if err != nil {
		t.Fatal(err)
	}
	if tr.TLSConfig == nil || len(tr.TLSConfig.Certificates) != 1 || tr.TLSConfig.RootCAs != nil {
		t.Fatalf("tls config = %+v", tr.TLSConfig)
	}

	tr, err = NewTransport(Opts{TLSCert: cert, TLSKey: key, TLSCA: cert})
	if err != nil {
		t.Fatal(err)
	}
	if tr.TLSConfig.RootCAs == nil {
		t.Fatal("CA pool not set")
	}
}

// Regression: --tls-ca was ignored without a client certificate.
func TestNewTransportCAOnly(t *testing.T) {
	cert, _ := selfSigned(t, t.TempDir())

	tr, err := NewTransport(Opts{TLSCA: cert})
	if err != nil {
		t.Fatal(err)
	}
	if tr.TLSConfig == nil || tr.TLSConfig.RootCAs == nil {
		t.Fatalf("CA not applied: %+v", tr.TLSConfig)
	}
	if len(tr.TLSConfig.Certificates) != 0 {
		t.Fatal("client certificate without one given")
	}

	if _, err := NewTransport(Opts{TLSCA: cert + ".missing"}); err == nil {
		t.Error("want CA read error")
	}
}

func TestNewTransportErrors(t *testing.T) {
	dir := t.TempDir()
	cert, key := selfSigned(t, dir)

	if _, err := NewTransport(Opts{TLSCert: filepath.Join(dir, "missing.pem"), TLSKey: key}); err == nil {
		t.Error("want keypair error")
	}
	if _, err := NewTransport(Opts{TLSCert: cert, TLSKey: key, TLSCA: filepath.Join(dir, "missing-ca.pem")}); err == nil {
		t.Error("want CA read error")
	}
}
