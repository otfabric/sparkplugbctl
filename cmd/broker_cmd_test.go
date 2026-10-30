// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/otfabric/go-sparkplugb/edge"
	"github.com/otfabric/go-sparkplugb/host"
	"github.com/otfabric/go-sparkplugb/mqtt"
	"github.com/otfabric/go-sparkplugb/payload"
	pb "github.com/otfabric/go-sparkplugb/proto"
)

const deadBroker = "tcp://127.0.0.1:1"

func rawClient(t *testing.T, url, id string) *mqtt.PahoTransport {
	t.Helper()
	tr := mqtt.NewPahoTransport()
	if err := tr.Connect(context.Background(), mqtt.ConnectOpts{BrokerURL: url, ClientID: id, CleanSession: true}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tr.Disconnect(context.Background()) })
	return tr
}

func encodeMetric(t *testing.T, name string) []byte {
	t.Helper()
	m, _ := payload.NewMetric(name, pb.DataType_Int32, int32(1))
	b, err := payload.Encode(payload.NewPayload(0, m))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func driveEdgeLifecycle(t *testing.T, url, group, node string) {
	t.Helper()
	ctx := context.Background()
	n, err := edge.New(mqtt.NewPahoTransport(), edge.Config{GroupID: group, EdgeNodeID: node, BrokerURL: url})
	if err != nil {
		t.Fatal(err)
	}
	if err := n.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	m, _ := payload.NewMetric("temp", pb.DataType_Double, 1.0)
	steps := []func() error{
		func() error { return n.PublishBirth(ctx, []*pb.Payload_Metric{m}) },
		func() error { return n.PublishDeviceBirth(ctx, "d1", []*pb.Payload_Metric{m}) },
		func() error { return n.PublishNodeData(ctx, []*pb.Payload_Metric{m}) },
		func() error { return n.PublishDeviceData(ctx, "d1", []*pb.Payload_Metric{m}) },
		func() error { return n.PublishDeviceDeath(ctx, "d1") },
		func() error { return n.Disconnect(ctx) },
	}
	for _, s := range steps {
		if err := s(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPublish(t *testing.T) {
	url := startBroker(t)
	sub := rawClient(t, url, "sub")
	var mu sync.Mutex
	var got []byte
	if err := sub.Subscribe(context.Background(), "spBv1.0/g/NDATA/n", 0, func(_ string, b []byte) {
		mu.Lock()
		got = b
		mu.Unlock()
	}); err != nil {
		t.Fatal(err)
	}

	out, _, err := run(t, nil, "publish", "--broker", url, "--topic", "spBv1.0/g/NDATA/n", "--seq", "3", "--metric", "a:Int32=5")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Published 1 metric(s) to spBv1.0/g/NDATA/n") {
		t.Fatalf("stdout = %q", out)
	}
	eventually(t, "published message", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return got != nil
	})
	mu.Lock()
	p, err := payload.Decode(got)
	mu.Unlock()
	if err != nil || p.GetSeq() != 3 {
		t.Fatalf("received seq=%d err=%v", p.GetSeq(), err)
	}

	if _, _, err := run(t, nil, "publish", "--broker", url, "--client-id", "custom", "--topic", "x/y", "--metric", "a:Int32=1"); err != nil {
		t.Fatalf("explicit client id: %v", err)
	}
}

func TestPublishErrors(t *testing.T) {
	cases := map[string][]string{
		"missing topic": {"publish"},
		"bad metric":    {"publish", "--topic", "t", "--metric", "bad"},
		"tls":           {"publish", "--topic", "t", "--tls-cert", "/nope.crt", "--tls-key", "/nope.key"},
		"connect":       {"publish", "--topic", "t", "--broker", deadBroker},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, err := run(t, nil, args...); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

func TestListen(t *testing.T) {
	url := startBroker(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := start(ctx, nil, "listen", "--broker", url, "--group", "g", "--node", "n", "--device", "d")
	eventually(t, "listening", contains(r.stderr, "Listening on spBv1.0/#"))

	pub := rawClient(t, url, "pub")
	send := func(topic string, data []byte) {
		if err := pub.Publish(context.Background(), topic, 0, false, data); err != nil {
			t.Fatal(err)
		}
	}
	send("spBv1.0/bad", encodeMetric(t, "unparseable"))
	send("spBv1.0/other/DDATA/n/d", encodeMetric(t, "othergroup"))
	send("spBv1.0/g/DDATA/other/d", encodeMetric(t, "othernode"))
	send("spBv1.0/g/DDATA/n/other", encodeMetric(t, "otherdevice"))
	send("spBv1.0/g/DDATA/n/d", []byte{0xff, 0xff, 0xff})
	send("spBv1.0/g/DDATA/n/d", encodeMetric(t, "marker"))

	eventually(t, "matching message", contains(r.stdout, "marker"))
	eventually(t, "decode error", contains(r.stderr, "decode error on spBv1.0/g/DDATA/n/d"))
	for _, unwanted := range []string{"unparseable", "othergroup", "othernode", "otherdevice"} {
		if strings.Contains(r.stdout.String(), unwanted) {
			t.Errorf("filtered message %q was printed", unwanted)
		}
	}

	cancel()
	if err := r.wait(t); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.stderr.String(), "Shutting down") {
		t.Errorf("stderr = %q", r.stderr.String())
	}
}

// TestListenState checks that listen prints a STATE message (JSON) and
// reports one that is not a STATE payload.
func TestListenState(t *testing.T) {
	url := startBroker(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := start(ctx, nil, "listen", "--broker", url)
	eventually(t, "listening", contains(r.stderr, "Listening on spBv1.0/#"))

	pub := rawClient(t, url, "pub")
	for _, data := range []string{`{"online":true,"timestamp":1767225600000}`, `not json`} {
		if err := pub.Publish(context.Background(), "spBv1.0/STATE/scada", 0, false, []byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	eventually(t, "STATE", contains(r.stdout, "[STATE] spBv1.0/STATE/scada\n  online=true  ts=2026-01-01T00:00:00Z"))
	eventually(t, "bad STATE", contains(r.stderr, "decode error on spBv1.0/STATE/scada: STATE payload"))

	cancel()
	if err := r.wait(t); err != nil {
		t.Fatal(err)
	}
}

func TestListenErrors(t *testing.T) {
	if _, _, err := run(t, nil, "listen", "--tls-cert", "/nope.crt", "--tls-key", "/nope.key"); err == nil {
		t.Error("want TLS error")
	}
	if _, _, err := run(t, nil, "listen", "--broker", deadBroker, "--client-id", "x"); err == nil {
		t.Error("want connect error")
	}
}

func TestHost(t *testing.T) {
	url := startBroker(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := start(ctx, nil, "host", "--broker", url, "--host-id", "h1")
	eventually(t, "host online", contains(r.stderr, "Listening for edge activity"))

	driveEdgeLifecycle(t, url, "g", "n")
	for _, want := range []string{"[NBIRTH] g/n", "[DBIRTH] g/n/d1", "[NDATA]  g/n", "[DDATA]  g/n/d1", "[DDEATH] g/n/d1", "[NDEATH] g/n"} {
		eventually(t, want, contains(r.stdout, want))
	}

	cancel()
	if err := r.wait(t); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.stderr.String(), "Disconnected") {
		t.Errorf("stderr = %q", r.stderr.String())
	}
}

func TestHostErrors(t *testing.T) {
	cases := map[string][]string{
		"missing host-id": {"host"},
		"tls":             {"host", "--host-id", "h", "--tls-cert", "/nope.crt", "--tls-key", "/nope.key"},
		"connect":         {"host", "--host-id", "h", "--client-id", "c", "--broker", deadBroker},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, err := run(t, nil, args...); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

func TestApp(t *testing.T) {
	all := []string{"[NBIRTH] g/n", "[DBIRTH] g/n/d1", "[NDATA]  g/n", "[DDATA]  g/n/d1", "[DDEATH] g/n/d1", "[NDEATH] g/n"}
	jsonTypes := []string{`"type":"NBIRTH"`, `"type":"DBIRTH"`, `"type":"NDATA"`, `"type":"DDATA"`, `"type":"DDEATH"`, `"type":"NDEATH"`}
	cases := []struct {
		name  string
		args  []string
		want  []string
		empty bool
	}{
		{"text", nil, all, false},
		{"json", []string{"--json"}, jsonTypes, false},
		{"matching group", []string{"--group", "g"}, all, false},
		{"filtered group", []string{"--group", "other"}, nil, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			url := startBroker(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			args := append([]string{"app", "--broker", url}, c.args...)
			r := start(ctx, nil, args...)
			eventually(t, "app listening", contains(r.stderr, "Listening for edge activity"))

			driveEdgeLifecycle(t, url, "g", "n")
			if c.empty {
				driveEdgeLifecycle(t, url, "other", "n")
				eventually(t, "unfiltered group", contains(r.stdout, "[NDEATH] other/n"))
			}
			for _, want := range c.want {
				eventually(t, want, contains(r.stdout, want))
			}

			cancel()
			if err := r.wait(t); err != nil {
				t.Fatal(err)
			}
			if c.empty && strings.Contains(r.stdout.String(), " g/n") {
				t.Errorf("filtered app printed group g:\n%s", r.stdout.String())
			}
			if c.empty && !strings.Contains(r.stderr.String(), "Filtering: group=other") {
				t.Errorf("stderr = %q", r.stderr.String())
			}
		})
	}
}

func TestAppErrors(t *testing.T) {
	if _, _, err := run(t, nil, "app", "--tls-cert", "/nope.crt", "--tls-key", "/nope.key"); err == nil {
		t.Error("want TLS error")
	}
	if _, _, err := run(t, nil, "app", "--broker", deadBroker, "--client-id", "c"); err == nil {
		t.Error("want connect error")
	}
}

func TestEdge(t *testing.T) {
	url := startBroker(t)
	bg := context.Background()
	h, err := host.New(mqtt.NewPahoTransport(), host.Config{HostID: "obs", BrokerURL: url})
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Connect(bg); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Close(bg) }()

	ctx, cancel := context.WithCancel(bg)
	defer cancel()
	r := start(ctx, nil, "edge", "--broker", url, "--group", "g", "--node", "n", "--device", "d1",
		"--interval", "20ms", "--metric", "a:Double=1", "--metric", "b:Float=2", "--metric", "c:Int32=3", "--metric", "s:String=x")
	eventually(t, "edge publishing", contains(r.stderr, "Published NDATA (3 metrics)"))
	if !strings.Contains(r.stderr.String(), "Published DBIRTH for d1") {
		t.Errorf("stderr = %q", r.stderr.String())
	}

	m, _ := payload.NewMetric("setpoint", pb.DataType_Int32, int32(1))
	if err := h.SendNodeCommand(bg, "g", "n", []*pb.Payload_Metric{m}); err != nil {
		t.Fatal(err)
	}
	if err := h.SendDeviceCommand(bg, "g", "n", "d1", []*pb.Payload_Metric{m}); err != nil {
		t.Fatal(err)
	}
	eventually(t, "NCMD", contains(r.stderr, "NCMD received: 1 metric(s)"))
	eventually(t, "DCMD", contains(r.stderr, "DCMD for d1: 1 metric(s)"))

	cancel()
	if err := r.wait(t); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Published DDEATH for d1", "Disconnected", "[INFO]"} {
		if !strings.Contains(r.stderr.String(), want) {
			t.Errorf("stderr missing %q", want)
		}
	}
}

// TestEdgePrimaryHost checks that the edge waits for its primary host: no
// birth or data until the host's STATE is online, then the births and data.
func TestEdgePrimaryHost(t *testing.T) {
	url := startBroker(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := start(ctx, nil, "edge", "--broker", url, "--group", "g", "--node", "n", "--device", "d1",
		"--interval", "20ms", "--primary-host", "scada")
	eventually(t, "waiting", contains(r.stderr, "Waiting for primary host scada to be online"))
	time.Sleep(100 * time.Millisecond)
	if s := r.stderr.String(); strings.Contains(s, "Published") || strings.Contains(s, "publish NBIRTH") {
		t.Fatalf("published before the primary host was online:\n%s", s)
	}

	pub := rawClient(t, url, "scada")
	if err := pub.Publish(context.Background(), "spBv1.0/STATE/scada", 1, true, []byte(`{"online":true,"timestamp":1}`)); err != nil {
		t.Fatal(err)
	}
	eventually(t, "births", contains(r.stderr, "Published DBIRTH for d1"))
	eventually(t, "data", contains(r.stderr, "Published NDATA"))

	cancel()
	if err := r.wait(t); err != nil {
		t.Fatal(err)
	}
}

func TestEdgeErrors(t *testing.T) {
	url := startBroker(t)
	cases := map[string][]string{
		"missing group": {"edge", "--node", "n"},
		"tls":           {"edge", "--group", "g", "--node", "n", "--tls-cert", "/nope.crt", "--tls-key", "/nope.key"},
		"connect":       {"edge", "--group", "g", "--node", "n", "--client-id", "c", "--broker", deadBroker},
		"bad metric":    {"edge", "--group", "g", "--node", "n", "--broker", url, "--metric", "bad"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, err := run(t, nil, args...); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

// Empty IDs pass the flags, and are refused as invalid input.
func TestFailureExitCodes(t *testing.T) {
	url := startBroker(t)
	for name, c := range map[string]struct {
		args []string
		want int
	}{
		"edge, empty group":   {[]string{"edge", "--group", "", "--node", "n", "--broker", url}, ExitInvalidInput},
		"host, empty host-id": {[]string{"host", "--host-id", "", "--broker", url}, ExitInvalidInput},
		"app, empty broker":   {[]string{"app", "--broker", ""}, ExitInvalidInput},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := run(t, nil, c.args...)
			if got := ExitCode(err); got != c.want {
				t.Errorf("exit %d, want %d (%v)", got, c.want, err)
			}
		})
	}
}

// A broker that accepts the connection and drops it fails the subscription
// of listen, whether the drop is seen before or during the subscribe.
func TestListenSubscribeFails(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		_, _ = c.Read(make([]byte, 1024))              // CONNECT
		_, _ = c.Write([]byte{0x20, 0x02, 0x00, 0x00}) // CONNACK, accepted
	}()
	_, _, err = run(t, nil, "listen", "--broker", "tcp://"+ln.Addr().String(), "--client-id", "dropped")
	if got := ExitCode(err); got != ExitTransport {
		t.Errorf("exit %d, want %d (%v)", got, ExitTransport, err)
	}
}
