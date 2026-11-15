// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/otfabric/sparkplugbctl/internal/broker"
)

// maxPasswordBytes is the longest password MQTT can carry (a two-byte
// length, MQTT 3.1.1 section 1.5.3).
const maxPasswordBytes = 65535

// tlsSchemes are the broker URL schemes the MQTT client (Paho) connects with
// TLS. With any other scheme it does not use the --tls-* files.
var tlsSchemes = map[string]bool{"ssl": true, "tls": true, "mqtts": true, "mqtt+ssl": true, "tcps": true, "wss": true}

// parseBrokerURL parses a broker URL as the MQTT client does: one without
// "://" is a tcp:// URL, and one that starts with ":" is on 127.0.0.1.
func parseBrokerURL(raw string) (*url.URL, error) {
	if strings.HasPrefix(raw, ":") {
		raw = "127.0.0.1" + raw
	}
	if !strings.Contains(raw, "://") {
		raw = "tcp://" + raw
	}
	return url.Parse(raw)
}

// redactURL returns the broker URL for output, with the password of its
// user information replaced by "xxxxx". A URL that does not parse is not
// printed: the MQTT client does not connect to it either.
func redactURL(raw string) string {
	u, err := parseBrokerURL(raw)
	if err != nil {
		return "(broker URL that does not parse)"
	}
	if u.User == nil {
		return raw
	}
	return u.Redacted()
}

// connOpts collects what a network command connects with: the broker URL,
// the TLS files and the login. It reads --password-file, and warns on stderr
// about a --tls-* flag that has no effect with the broker URL's scheme and
// about a password without a username, which MQTT cannot send.
func connOpts(stderr io.Writer) (broker.Opts, error) {
	pw := password
	if passwordFile != "" {
		var err error
		if pw, err = readPassword(passwordFile); err != nil {
			return broker.Opts{}, withExit(ExitInvalidInput, err)
		}
	}
	if pw != "" && username == "" {
		_, _ = fmt.Fprintln(stderr, "warning: the password is not sent without --username: MQTT carries a password only with a username")
		pw = ""
	}
	warnTLSScheme(stderr)
	return broker.Opts{
		BrokerURL: brokerURL,
		ClientID:  clientID,
		Username:  username,
		Password:  pw,
		TLSCert:   tlsCert,
		TLSKey:    tlsKey,
		TLSCA:     tlsCA,
	}, nil
}

// warnTLSScheme writes a warning on stderr for each --tls-* flag that is set
// while the broker URL's scheme connects without TLS.
func warnTLSScheme(stderr io.Writer) {
	var set []string
	for _, f := range []struct{ name, value string }{{"tls-cert", tlsCert}, {"tls-key", tlsKey}, {"tls-ca", tlsCA}} {
		if f.value != "" {
			set = append(set, f.name)
		}
	}
	if len(set) == 0 {
		return
	}
	u, err := parseBrokerURL(brokerURL)
	if err != nil || tlsSchemes[u.Scheme] {
		return
	}
	for _, name := range set {
		_, _ = fmt.Fprintf(stderr, "warning: --%s has no effect with a %s:// broker URL, which connects without TLS; TLS needs tls://, ssl://, mqtts:// or wss://\n", name, u.Scheme)
	}
}

// readPassword reads the password of --password-file: the whole file but
// one trailing line end (\n or \r\n).
func readPassword(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("password file: %w", err)
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, maxPasswordBytes+3)) // the password and \r\n, and one more
	if err != nil {
		return "", fmt.Errorf("password file: %w", err)
	}
	b = bytes.TrimSuffix(b, []byte("\n"))
	b = bytes.TrimSuffix(b, []byte("\r"))
	if len(b) > maxPasswordBytes {
		return "", fmt.Errorf("password file %s: longer than the %d bytes MQTT allows", path, maxPasswordBytes)
	}
	if len(b) == 0 {
		return "", errors.New("password file " + path + " is empty")
	}
	return string(b), nil
}
