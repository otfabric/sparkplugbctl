// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
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
