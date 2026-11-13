// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"regexp"
	"sync"
	"testing"

	"github.com/otfabric/go-sparkplugb/mqtt"
)

// clientIDTransport records the client ID of each Connect and fails it, so
// that every command returns right after connecting.
type clientIDTransport struct {
	faultTransport

	mu  sync.Mutex
	ids []string
}

func (c *clientIDTransport) Connect(_ context.Context, o mqtt.ConnectOpts) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ids = append(c.ids, o.ClientID)
	return errInjected
}

// clientIDOf runs args against a clientIDTransport and returns the client ID
// the command connected with.
func clientIDOf(t *testing.T, args ...string) string {
	t.Helper()
	tr := &clientIDTransport{}
	injectTransport(t, tr)
	if _, _, err := run(t, nil, args...); !errors.Is(err, errInjected) {
		t.Fatalf("%v: error = %v, want the injected connect fault", args, err)
	}
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if len(tr.ids) != 1 {
		t.Fatalf("%v: %d connects, want 1", args, len(tr.ids))
	}
	return tr.ids[0]
}

func TestRandomClientID(t *testing.T) {
	for _, command := range []string{"app", "listen", "publish"} {
		re := regexp.MustCompile(`^sparkplugbctl-` + command + `-[0-9a-f]{8}$`)
		seen := map[string]bool{}
		for range 64 {
			id := randomClientID(command)
			if !re.MatchString(id) {
				t.Fatalf("randomClientID(%q) = %q, want %s", command, id, re)
			}
			if seen[id] {
				t.Fatalf("randomClientID(%q) drew %q twice", command, id)
			}
			seen[id] = true
		}
	}
}

// TestDefaultClientIDs checks the client ID each command connects with: a
// random one for app, listen and publish, a new one each run, so that two runs
// against one broker do not take each other over [MQTT-3.1.4-2]; the Sparkplug
// identity for host and edge; and --client-id, or its variable, as given.
func TestDefaultClientIDs(t *testing.T) {
	random := map[string][]string{
		"app":     {"app"},
		"listen":  {"listen"},
		"publish": {"publish", "--topic", "t", "--metric", "a:Int32=1"},
	}
	for name, args := range random {
		t.Run(name, func(t *testing.T) {
			re := regexp.MustCompile(`^sparkplugbctl-` + name + `-[0-9a-f]{8}$`)
			first, second := clientIDOf(t, args...), clientIDOf(t, args...)
			for _, id := range []string{first, second} {
				if !re.MatchString(id) {
					t.Errorf("client ID %q, want %s", id, re)
				}
			}
			if first == second {
				t.Errorf("two runs both connected as %q", first)
			}
			if got := clientIDOf(t, append(args, "--client-id", "custom")...); got != "custom" {
				t.Errorf("with --client-id custom: client ID %q", got)
			}
			t.Setenv("SPARKPLUGBCTL_CLIENT_ID", "from-env")
			if got := clientIDOf(t, args...); got != "from-env" {
				t.Errorf("with SPARKPLUGBCTL_CLIENT_ID=from-env: client ID %q", got)
			}
		})
	}

	fixed := map[string]struct {
		args []string
		want string
	}{
		"host": {[]string{"host", "--host-id", "h1"}, "sparkplugbctl-host-h1"},
		"edge": {[]string{"edge", "--group", "g1", "--node", "n1"}, "sparkplugbctl-edge-g1-n1"},
	}
	for name, c := range fixed {
		t.Run(name, func(t *testing.T) {
			for range 2 {
				if got := clientIDOf(t, c.args...); got != c.want {
					t.Errorf("client ID %q, want %q", got, c.want)
				}
			}
			if got := clientIDOf(t, append(c.args, "--client-id", "custom")...); got != "custom" {
				t.Errorf("with --client-id custom: client ID %q", got)
			}
		})
	}
}
