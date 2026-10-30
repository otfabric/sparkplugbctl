// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"testing"

	"github.com/otfabric/go-sparkplugb/mqtt"
	pb "github.com/otfabric/go-sparkplugb/proto"
	"github.com/otfabric/sparkplugbctl/internal/broker"
)

var errInjected = errors.New("injected fault")

// faultTransport is an mqtt.Transport without a broker: every call succeeds
// except those given a fault.
type faultTransport struct {
	connectErr   error
	subscribeErr error
	failTopic    string // Publish to a topic with this substring fails with publishErr
	publishErr   error

	mu          sync.Mutex
	published   []string
	disconnects int
}

func (f *faultTransport) Connect(context.Context, mqtt.ConnectOpts) error { return f.connectErr }

func (f *faultTransport) Publish(_ context.Context, topic string, _ byte, _ bool, _ []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.publishErr != nil && strings.Contains(topic, f.failTopic) {
		return f.publishErr
	}
	f.published = append(f.published, topic)
	return nil
}

func (f *faultTransport) Subscribe(context.Context, string, byte, mqtt.MessageHandler) error {
	return f.subscribeErr
}

func (f *faultTransport) Disconnect(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.disconnects++
	return nil
}

func (f *faultTransport) disconnected() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.disconnects
}

func injectTransport(t *testing.T, tr mqtt.Transport) {
	t.Helper()
	old := newTransport
	newTransport = func(broker.Opts) (mqtt.Transport, error) { return tr, nil }
	t.Cleanup(func() { newTransport = old })
}

// TestConnectFaults checks that each command wraps a failed connection and
// that the exit code follows the cause.
func TestConnectFaults(t *testing.T) {
	commands := map[string][]string{
		"listen":  {"listen"},
		"publish": {"publish", "--topic", "t", "--metric", "a:Int32=1"},
		"host":    {"host", "--host-id", "h"},
		"app":     {"app"},
		"edge":    {"edge", "--group", "g", "--node", "n"},
	}
	causes := []struct {
		name string
		err  error
		want int
	}{
		{"refused", errInjected, ExitTransport},
		{"timeout", context.DeadlineExceeded, ExitTimeout},
		{"interrupted", context.Canceled, ExitInterrupted},
		{"permission", fs.ErrPermission, ExitPermission},
	}
	for name, args := range commands {
		for _, c := range causes {
			t.Run(name+"/"+c.name, func(t *testing.T) {
				injectTransport(t, &faultTransport{connectErr: c.err})
				_, _, err := run(t, nil, args...)
				if !errors.Is(err, c.err) {
					t.Fatalf("error = %v, want it to wrap %v", err, c.err)
				}
				if !strings.Contains(err.Error(), "connect") {
					t.Errorf("error = %v, want the connect context", err)
				}
				if got := ExitCode(err); got != c.want {
					t.Errorf("exit %d, want %d", got, c.want)
				}
			})
		}
	}
}

func TestListenSubscribeFault(t *testing.T) {
	tr := &faultTransport{subscribeErr: errInjected}
	injectTransport(t, tr)
	_, _, err := run(t, nil, "listen")
	if !errors.Is(err, errInjected) || !strings.HasPrefix(err.Error(), "subscribe: ") {
		t.Fatalf("error = %v", err)
	}
	if got := ExitCode(err); got != ExitTransport {
		t.Errorf("exit %d, want %d", got, ExitTransport)
	}
	if tr.disconnected() != 1 {
		t.Errorf("disconnects = %d, want 1", tr.disconnected())
	}
}

func TestPublishFault(t *testing.T) {
	tr := &faultTransport{publishErr: errInjected}
	injectTransport(t, tr)
	stdout, _, err := run(t, nil, "publish", "--topic", "t", "--metric", "a:Int32=1")
	if !errors.Is(err, errInjected) || !strings.HasPrefix(err.Error(), "publish: ") {
		t.Fatalf("error = %v", err)
	}
	if got := ExitCode(err); got != ExitTransport {
		t.Errorf("exit %d, want %d", got, ExitTransport)
	}
	if strings.Contains(stdout, "Published") {
		t.Errorf("stdout = %q, reported success", stdout)
	}
	if tr.disconnected() != 1 {
		t.Errorf("disconnects = %d, want 1", tr.disconnected())
	}
}

// TestEdgePublishFaults checks that a failed publish is reported and leaves
// the simulator running, and that it still shuts down cleanly.
func TestEdgePublishFaults(t *testing.T) {
	for name, c := range map[string]struct{ topic, want string }{
		"NBIRTH": {"/NBIRTH/", "publish NBIRTH: injected fault"},
		"DBIRTH": {"/DBIRTH/", "publish DBIRTH d1: injected fault"},
		"NDATA":  {"/NDATA/", "publish NDATA: injected fault"},
	} {
		t.Run(name, func(t *testing.T) {
			tr := &faultTransport{failTopic: c.topic, publishErr: errInjected}
			injectTransport(t, tr)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r := start(ctx, nil, "edge", "--group", "g", "--node", "n", "--device", "d1",
				"--interval", "10ms", "--metric", "a:Double=1")
			eventually(t, "the fault", contains(r.stderr, c.want))

			cancel()
			if err := r.wait(t); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(r.stderr.String(), "Disconnected") {
				t.Errorf("stderr = %q", r.stderr.String())
			}
		})
	}
}

func TestEncodeFault(t *testing.T) {
	old := encodePayload
	encodePayload = func(*pb.Payload) ([]byte, error) { return nil, fmt.Errorf("marshal: %w", errInjected) }
	t.Cleanup(func() { encodePayload = old })

	for name, args := range map[string][]string{
		"encode":  {"encode", "--metric", "a:Int32=1"},
		"publish": {"publish", "--topic", "t", "--metric", "a:Int32=1"},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := run(t, nil, args...)
			if !errors.Is(err, errInjected) || !strings.HasPrefix(err.Error(), "encode: ") {
				t.Fatalf("error = %v", err)
			}
			if got := ExitCode(err); got != ExitInvalidInput {
				t.Errorf("exit %d, want %d", got, ExitInvalidInput)
			}
		})
	}
}

// TestNewTransportError checks that a refused TLS setup reaches the command
// as invalid input through the seam, with no transport.
func TestNewTransportError(t *testing.T) {
	tr, err := newTransport(broker.Opts{TLSCert: "/nope.crt", TLSKey: "/nope.key"})
	if err == nil {
		t.Fatal("want error")
	}
	if tr != nil {
		t.Fatalf("transport = %v, want nil", tr)
	}
	_, _, err = run(t, nil, "listen", "--tls-cert", "/nope.crt", "--tls-key", "/nope.key")
	if got := ExitCode(err); got != ExitInvalidInput {
		t.Errorf("exit %d, want %d (%v)", got, ExitInvalidInput, err)
	}
}
