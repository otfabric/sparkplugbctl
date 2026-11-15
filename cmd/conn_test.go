// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// TestTLSFlagsWithoutTLSScheme checks that each --tls-* flag set with a
// broker URL whose scheme connects without TLS is named in a warning on
// stderr with that scheme, and that a TLS scheme gives no warning.
func TestTLSFlagsWithoutTLSScheme(t *testing.T) {
	defer func(b, c, k, a string) { brokerURL, tlsCert, tlsKey, tlsCA = b, c, k, a }(brokerURL, tlsCert, tlsKey, tlsCA)
	for _, c := range []struct {
		url, scheme string
		tls         bool
	}{
		{"tcp://broker:1883", "tcp", false},
		{"mqtt://broker:1883", "mqtt", false},
		{"TCP://broker:1883", "tcp", false},
		{"broker:1883", "tcp", false},
		{":1883", "tcp", false},
		{"ws://broker:9001/mqtt", "ws", false},
		{"unix:///run/mqtt.sock", "unix", false},
		{"tls://broker:8883", "tls", true},
		{"ssl://broker:8883", "ssl", true},
		{"mqtts://broker:8883", "mqtts", true},
		{"mqtt+ssl://broker:8883", "mqtt+ssl", true},
		{"tcps://broker:8883", "tcps", true},
		{"wss://broker:443/mqtt", "wss", true},
	} {
		for _, flags := range [][3]string{{"", "", "ca.pem"}, {"cert.pem", "key.pem", ""}, {"cert.pem", "", ""}, {"", "", ""}} {
			brokerURL, tlsCert, tlsKey, tlsCA = c.url, flags[0], flags[1], flags[2]
			var stderr bytes.Buffer
			warnTLSScheme(&stderr)
			for i, name := range []string{"tls-cert", "tls-key", "tls-ca"} {
				line := "warning: --" + name + " has no effect with a " + c.scheme + ":// broker URL, which connects without TLS"
				want := !c.tls && flags[i] != ""
				if got := strings.Contains(stderr.String(), line); got != want {
					t.Errorf("%s with %v: warning for --%s %t, want %t: %q", c.url, flags, name, got, want, stderr.String())
				}
			}
			if c.tls && stderr.Len() != 0 {
				t.Errorf("%s: %q", c.url, stderr.String())
			}
		}
	}
}

// TestTLSFlagsWarningOnCommand runs a command with --tls-ca and a tcp://
// broker: the warning is on stderr before it connects.
func TestTLSFlagsWarningOnCommand(t *testing.T) {
	url := startBroker(t)
	_, ca := startTLSBroker(t)
	_, stderr, err := run(t, nil, "publish", "--broker", url, "--tls-ca", ca, "--topic", "spBv1.0/g/NDATA/n", "--metric", "a:Int32=1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, "warning: --tls-ca has no effect with a tcp:// broker URL") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestRedactURL(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"tcp://localhost:1883", "tcp://localhost:1883"},
		{"localhost:1883", "localhost:1883"},
		{"tcp://user:secret@broker:1883", "tcp://user:xxxxx@broker:1883"},
		{"user:secret@broker:1883", "tcp://user:xxxxx@broker:1883"},
		{"tls://user@broker:8883", "tls://user@broker:8883"},
		{"ws://u:p%40ss@broker:9001/mqtt", "ws://u:xxxxx@broker:9001/mqtt"},
		{"tcp://u:s%zz@broker", "(broker URL that does not parse)"},
	} {
		if got := redactURL(c.in); got != c.want {
			t.Errorf("redactURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestBrokerURLPasswordNotPrinted runs the commands that print the broker
// URL with a password in its user information, which the MQTT client logs
// in with: stderr names the broker but not the password.
func TestBrokerURLPasswordNotPrinted(t *testing.T) {
	const user, pass = "operator", "url-secret-0815"
	url, hook := startAuthBroker(t, user, pass)
	withLogin := strings.Replace(url, "tcp://", "tcp://"+user+":"+pass+"@", 1)
	host := strings.TrimPrefix(url, "tcp://")
	for _, name := range []string{"listen", "app", "host", "edge"} {
		t.Run(name, func(t *testing.T) {
			hook.logins()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r := start(ctx, nil, append(networkCommands[name].args, "--broker", withLogin)...)
			eventually(t, "connected", contains(r.stderr, networkCommands[name].ready))
			cancel()
			if err := r.wait(t); err != nil {
				t.Fatal(err)
			}
			if logins := hook.logins(); len(logins) == 0 || logins[0] != user {
				t.Fatalf("broker saw logins %q: the URL's user information did not log in", logins)
			}
			out := r.stdout.String() + r.stderr.String()
			if strings.Contains(out, pass) {
				t.Errorf("the password is in the output:\n%s", out)
			}
			if !strings.Contains(out, "tcp://"+user+":xxxxx@"+host) {
				t.Errorf("the broker is not named:\n%s", out)
			}
		})
	}
}
