// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/otfabric/go-sparkplugb/payload"
)

// TestEdgeValidatesBeforeConnect checks that edge refuses a metric that does
// not parse and a publish interval that is not positive before it connects:
// the transport sees no Connect, so the broker holds no NDEATH will of the
// node to publish.
func TestEdgeValidatesBeforeConnect(t *testing.T) {
	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		{"bad metric", []string{"--metric", "bad"}, `metric "bad"`},
		{"bad metric type", []string{"--metric", "a:Nope=1"}, `metric "a:Nope=1"`},
		{"interval 0", []string{"--interval", "0s"}, "--interval 0s: must be more than 0"},
		{"interval -1ns", []string{"--interval", "-1ns"}, "--interval -1ns: must be more than 0"},
		{"interval -1s", []string{"--interval", "-1s"}, "--interval -1s: must be more than 0"},
	} {
		t.Run(c.name, func(t *testing.T) {
			tr := &recTransport{}
			injectTransport(t, tr)
			_, _, err := run(t, nil, append([]string{"edge", "--group", "g", "--node", "n"}, c.args...)...)
			if got := ExitCode(err); got != ExitInvalidInput {
				t.Errorf("exit %d, want %d (%v)", got, ExitInvalidInput, err)
			}
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %v, want %q", err, c.want)
			}
			if calls := tr.callList(); len(calls) != 0 {
				t.Errorf("transport calls %v, want none", calls)
			}
		})
	}

	// The smallest interval is accepted.
	tr := &recTransport{}
	injectTransport(t, tr)
	injectTicker(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := start(ctx, nil, "edge", "--group", "g", "--node", "n", "--interval", "1ns")
	eventually(t, "running", contains(r.stderr, "Publishing data every 1ns"))
	cancel()
	if err := r.wait(t); err != nil {
		t.Fatal(err)
	}
	if tr.count("Connect") != 1 || tr.count("Disconnect") != 1 {
		t.Errorf("calls %v, want one Connect and one Disconnect", tr.callList())
	}
}

// TestEdgeChecksIDsBeforeConnect checks that edge refuses a group, edge node
// or device ID that cannot be a segment of a Sparkplug topic (empty, not
// valid UTF-8, or with '/', '+' or '#') as invalid input, with a message that
// names the flag and the quoted value, before it connects: the transport
// sees no Connect.
// An accepted command line reaches the Connect, which the transport fails,
// so that the command ends at once.
func TestEdgeChecksIDsBeforeConnect(t *testing.T) {
	const rule = ": a Sparkplug ID is non-empty valid UTF-8 without '/', '+' or '#'"
	for _, c := range []struct {
		name        string
		group, node string
		devices     []string // one --device flag each
		want        string   // the flag and the value as the message quotes them; empty when accepted
	}{
		{"one character each", "g", "n", []string{"d"}, ""},
		{"one multi-byte character", "é", "ñ", []string{"ü"}, ""},
		{"no device", "plant", "gw1", nil, ""},
		{"two devices", "plant", "gw1", []string{"sensor1", "sensor2"}, ""},
		// pflag reads `--device ""` as an empty list: no device ID is given.
		{"empty device flag", "g", "n", []string{""}, ""},

		{"group empty", "", "n", nil, `--group ""`},
		{"group /", "/", "n", nil, `--group "/"`},
		{"group with /", "a/b", "n", nil, `--group "a/b"`},
		{"group +", "+", "n", nil, `--group "+"`},
		{"group with +", "a+b", "n", nil, `--group "a+b"`},
		{"group #", "#", "n", nil, `--group "#"`},
		{"group with #", "a#b", "n", nil, `--group "a#b"`},
		{"group not UTF-8", "\xff", "n", nil, `--group "\xff"`},

		{"node empty", "g", "", nil, `--node ""`},
		{"node /", "g", "/", nil, `--node "/"`},
		{"node with /", "g", "a/b", nil, `--node "a/b"`},
		{"node +", "g", "+", nil, `--node "+"`},
		{"node with +", "g", "a+b", nil, `--node "a+b"`},
		{"node #", "g", "#", nil, `--node "#"`},
		{"node with #", "g", "a#b", nil, `--node "a#b"`},
		{"node with a byte that is not UTF-8", "g", "a\xffb", nil, `--node "a\xffb"`},

		{"device empty in a list", "g", "n", []string{"d1,"}, `--device ""`},
		{"device /", "g", "n", []string{"/"}, `--device "/"`},
		{"device with /", "g", "n", []string{"a/b"}, `--device "a/b"`},
		{"device +", "g", "n", []string{"+"}, `--device "+"`},
		{"device with +", "g", "n", []string{"a+b"}, `--device "a+b"`},
		{"device #", "g", "n", []string{"#"}, `--device "#"`},
		{"device with #", "g", "n", []string{"a#b"}, `--device "a#b"`},
		{"device not UTF-8", "g", "n", []string{"\xff"}, `--device "\xff"`},
		{"device with a byte that is not UTF-8", "g", "n", []string{"a\xffb"}, `--device "a\xffb"`},
		{"device with a control character", "g", "n", []string{"a\x1b/b"}, `--device "a\x1b/b"`},
		{"second device", "g", "n", []string{"d1", "d/2"}, `--device "d/2"`},
	} {
		t.Run(c.name, func(t *testing.T) {
			tr := &recTransport{connectErr: errInjected}
			injectTransport(t, tr)
			args := []string{"edge", "--group", c.group, "--node", c.node}
			for _, d := range c.devices {
				args = append(args, "--device", d)
			}
			_, _, err := run(t, nil, args...)
			if c.want == "" {
				if !errors.Is(err, errInjected) || ExitCode(err) != ExitTransport {
					t.Errorf("error %v (exit %d), want the injected connect error", err, ExitCode(err))
				}
				if n := tr.count("Connect"); n != 1 {
					t.Errorf("transport calls %v, want one Connect", tr.callList())
				}
				return
			}
			if got := ExitCode(err); got != ExitInvalidInput {
				t.Errorf("exit %d, want %d (%v)", got, ExitInvalidInput, err)
			}
			if err == nil || !strings.Contains(err.Error(), c.want+rule) {
				t.Errorf("error %v, want %q", err, c.want+rule)
			}
			if calls := tr.callList(); len(calls) != 0 {
				t.Errorf("transport calls %v, want none", calls)
			}
		})
	}
}

// TestPublishValidatesBeforeConnect checks that publish refuses a topic
// with a wildcard and a sequence number above 255 before it connects, and
// sends the bounds 0 and 255.
func TestPublishValidatesBeforeConnect(t *testing.T) {
	for _, c := range []struct {
		name  string
		topic string
		seq   string
		ok    bool
		want  string
	}{
		{"seq 0", "spBv1.0/g/NDATA/n", "0", true, ""},
		{"seq 1", "spBv1.0/g/NDATA/n", "1", true, ""},
		{"seq 254", "spBv1.0/g/NDATA/n", "254", true, ""},
		{"seq 255", "spBv1.0/g/NDATA/n", "255", true, ""},
		{"seq 256", "spBv1.0/g/NDATA/n", "256", false, "--seq 256: a Sparkplug sequence number is 0 to 255"},
		{"seq 2^32", "spBv1.0/g/NDATA/n", "4294967296", false, "--seq 4294967296"},
		{"seq max", "spBv1.0/g/NDATA/n", "18446744073709551615", false, "--seq 18446744073709551615"},
		{"multi-level wildcard", "spBv1.0/g/NDATA/#", "0", false, "has no wildcard"},
		{"single-level wildcard", "spBv1.0/+/NDATA/n", "0", false, "has no wildcard"},
		{"wildcard in a segment", "spBv1.0/g/NDATA/n+1", "0", false, "has no wildcard"},
		{"only #", "#", "0", false, "has no wildcard"},
		{"other topic", "plant/line1/raw", "0", true, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			tr := &recTransport{}
			injectTransport(t, tr)
			_, _, err := run(t, nil, "publish", "--topic", c.topic, "--seq", c.seq, "--metric", "a:Int32=1")
			if !c.ok {
				if got := ExitCode(err); got != ExitInvalidInput {
					t.Errorf("exit %d, want %d (%v)", got, ExitInvalidInput, err)
				}
				if err == nil || !strings.Contains(err.Error(), c.want) {
					t.Errorf("error %v, want %q", err, c.want)
				}
				if calls := tr.callList(); len(calls) != 0 {
					t.Errorf("transport calls %v, want none", calls)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			tr.mu.Lock()
			msgs := tr.published[c.topic]
			tr.mu.Unlock()
			if len(msgs) != 1 {
				t.Fatalf("published %v", tr.callList())
			}
			p, err := payload.Decode(msgs[0])
			if err != nil || c.seq != strconv.FormatUint(p.GetSeq(), 10) {
				t.Errorf("published seq %d (%v), want %s", p.GetSeq(), err, c.seq)
			}
		})
	}
}
