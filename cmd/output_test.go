// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"io/fs"
	"strings"
	"syscall"
	"testing"

	"github.com/otfabric/go-sparkplugb/host"
	"github.com/otfabric/go-sparkplugb/mqtt"
	"github.com/otfabric/go-sparkplugb/payload"
	pb "github.com/otfabric/go-sparkplugb/proto"
)

// fullStdout fails every write as a file on a full disk does.
type fullStdout struct{}

func (fullStdout) Write([]byte) (int, error) {
	return 0, &fs.PathError{Op: "write", Path: "/dev/stdout", Err: syscall.ENOSPC}
}

// TestOutputWriteErrors runs every command that writes its results to
// stdout with a stdout that cannot be written: each exits 10, "the output
// could not be written" (docs/exitcodes.md).
func TestOutputWriteErrors(t *testing.T) {
	encoded, _, err := run(t, nil, "encode", "--metric", "a:Int32=1")
	if err != nil {
		t.Fatal(err)
	}
	for name, args := range map[string][]string{
		"decode":         {"decode"},
		"decode --json":  {"decode", "--json"},
		"version":        {"version"},
		"version --json": {"version", "--json"},
		"encode":         {"encode", "--metric", "a:Int32=1"},
		"publish":        {"publish", "--topic", "spBv1.0/g/NDATA/n", "--metric", "a:Int32=1"},
	} {
		t.Run(name, func(t *testing.T) {
			injectTransport(t, &recTransport{})
			r := startOut(context.Background(), strings.NewReader(encoded), fullStdout{}, args...)
			err := r.wait(t)
			if got := ExitCode(err); got != ExitInternal {
				t.Errorf("exit %d, want %d (%v)", got, ExitInternal, err)
			}
			if err == nil || !strings.Contains(err.Error(), "write output") {
				t.Errorf("error %v, want it to name the output", err)
			}
		})
	}
}

// TestStreamWriteErrors runs listen, host and app with a stdout that cannot
// be written: the first message they print ends them with exit 10, after
// they disconnect.
func TestStreamWriteErrors(t *testing.T) {
	t.Run("listen", func(t *testing.T) {
		for _, msg := range []struct{ topic, data string }{
			{"spBv1.0/g/NDATA/n", string(encodeMetric(t, "m"))},
			{"spBv1.0/STATE/h", `{"online":true}`},
		} {
			tr := &recTransport{}
			injectTransport(t, tr)
			r := startOut(context.Background(), nil, fullStdout{}, "listen")
			eventually(t, "listening", contains(r.stderr, "Listening on"))
			tr.deliver(t, "spBv1.0/#", msg.topic, []byte(msg.data))
			err := r.wait(t)
			if got := ExitCode(err); got != ExitInternal {
				t.Errorf("%s: exit %d, want %d (%v)", msg.topic, got, ExitInternal, err)
			}
			if tr.count("Disconnect") != 1 {
				t.Errorf("%s: calls %v, want a Disconnect", msg.topic, tr.callList())
			}
		}
	})
	for _, args := range [][]string{{"host", "--host-id", "h"}, {"app"}, {"app", "--json"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			url := startBroker(t)
			r := startOut(context.Background(), nil, fullStdout{}, append(args, "--broker", url)...)
			eventually(t, "listening", contains(r.stderr, "Listening for edge activity"))
			driveEdgeLifecycle(t, url, "g", "n")
			err := r.wait(t)
			if got := ExitCode(err); got != ExitInternal {
				t.Errorf("exit %d, want %d (%v)", got, ExitInternal, err)
			}
			if !strings.Contains(r.stderr.String(), "Disconnected") {
				t.Errorf("stderr = %q", r.stderr.String())
			}
		})
	}
}

// TestEdgeDeviceDeathFailure checks that edge reports a DDEATH that could
// not be published as such, and not as published.
func TestEdgeDeviceDeathFailure(t *testing.T) {
	tr := &recTransport{failTopic: "/DDEATH/", publishErr: errInjected}
	injectTransport(t, tr)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := start(ctx, nil, "edge", "--group", "g", "--node", "n", "--device", "d1", "--device", "d2", "--interval", "1h")
	eventually(t, "births", contains(r.stderr, "Published DBIRTH for d2"))
	cancel()
	if err := r.wait(t); err != nil {
		t.Fatal(err)
	}
	stderr := r.stderr.String()
	for _, d := range []string{"d1", "d2"} {
		if strings.Contains(stderr, "Published DDEATH for "+d) {
			t.Errorf("reported a DDEATH of %s that failed as published:\n%s", d, stderr)
		}
		if !strings.Contains(stderr, "publish DDEATH "+d+": "+errInjected.Error()) {
			t.Errorf("no report of the failed DDEATH of %s:\n%s", d, stderr)
		}
	}
}

// TestTextOutputEscapesBrokerStrings checks that host, app, listen and edge
// print group, edge node and device IDs that other clients chose with their
// control and bidirectional formatting characters escaped.
func TestTextOutputEscapesBrokerStrings(t *testing.T) {
	const group, node = "g\x1b[2J", "n\u202e"
	const escGroup, escNode = `g\x1b[2J`, `n\u202e`
	for _, args := range [][]string{{"host", "--host-id", "h"}, {"app"}} {
		t.Run(args[0], func(t *testing.T) {
			url := startBroker(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r := start(ctx, nil, append(args, "--broker", url)...)
			eventually(t, "listening", contains(r.stderr, "Listening for edge activity"))
			driveEdgeLifecycle(t, url, group, node)
			eventually(t, "NDEATH", contains(r.stdout, "[NDEATH] "+escGroup+"/"+escNode))
			cancel()
			if err := r.wait(t); err != nil {
				t.Fatal(err)
			}
			out := r.stdout.String()
			for _, want := range []string{"[NBIRTH] " + escGroup + "/" + escNode, "[DDATA]  " + escGroup + "/" + escNode + "/d1"} {
				if !strings.Contains(out, want) {
					t.Errorf("stdout lacks %q:\n%s", want, out)
				}
			}
			if strings.ContainsAny(out, "\x1b\u202e") {
				t.Errorf("stdout has a raw control character:\n%q", out)
			}
		})
	}

	t.Run("listen decode error", func(t *testing.T) {
		tr := &recTransport{}
		injectTransport(t, tr)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		r := start(ctx, nil, "listen")
		eventually(t, "listening", contains(r.stderr, "Listening on"))
		tr.deliver(t, "spBv1.0/#", "spBv1.0/"+group+"/NDATA/"+node, []byte{0xff, 0xff})
		eventually(t, "decode error", contains(r.stderr, "decode error on spBv1.0/"+escGroup+"/NDATA/"+escNode+": "))
		cancel()
		if err := r.wait(t); err != nil {
			t.Fatal(err)
		}
		if strings.ContainsAny(r.stderr.String(), "\x1b\u202e") {
			t.Errorf("stderr has a raw control character:\n%q", r.stderr.String())
		}
	})

	t.Run("edge DCMD", func(t *testing.T) {
		url := startBroker(t)
		h, err := host.New(mqtt.NewPahoTransport(), host.Config{HostID: "obs", BrokerURL: url})
		if err != nil {
			t.Fatal(err)
		}
		bg := context.Background()
		if err := h.Connect(bg); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = h.Close(bg) }()
		ctx, cancel := context.WithCancel(bg)
		defer cancel()
		r := start(ctx, nil, "edge", "--broker", url, "--group", "g", "--node", "n", "--interval", "1h")
		eventually(t, "online", contains(r.stderr, "Published NBIRTH"))
		m, _ := payload.NewMetric("setpoint", pb.DataType_Int32, int32(1))
		if err := h.SendDeviceCommand(bg, "g", "n", "d\x1b]0;x\a", []*pb.Payload_Metric{m}); err != nil {
			t.Fatal(err)
		}
		eventually(t, "DCMD", contains(r.stderr, `DCMD for d\x1b]0;x\a: 1 metric(s)`))
		cancel()
		if err := r.wait(t); err != nil {
			t.Fatal(err)
		}
	})
}
