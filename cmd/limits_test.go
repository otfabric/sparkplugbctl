// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/otfabric/go-sparkplugb/payload"
	pb "github.com/otfabric/go-sparkplugb/proto"
)

// payloadOfSize returns an encoded payload of exactly n bytes: one String
// metric filled to the size.
func payloadOfSize(t *testing.T, n int) []byte {
	t.Helper()
	fill := n
	for range 8 {
		m, err := payload.NewMetric("s", pb.DataType_String, strings.Repeat("a", max(fill, 0)))
		if err != nil {
			t.Fatal(err)
		}
		b, err := payload.Encode(&pb.Payload{Seq: new(uint64), Metrics: []*pb.Payload_Metric{m}})
		if err != nil {
			t.Fatal(err)
		}
		if len(b) == n {
			return b
		}
		fill += n - len(b)
	}
	t.Fatalf("no payload of %d bytes", n)
	return nil
}

// TestDecodeMaxPayload checks the bound of decode at --max-payload and at
// the default: a payload of the bound decodes, one byte more is invalid
// input (exit 3).
func TestDecodeMaxPayload(t *testing.T) {
	const size = 300
	data := payloadOfSize(t, size)
	for _, c := range []struct {
		limit uint64
		ok    bool
	}{
		{size - 1, false},
		{size, true},
		{size + 1, true},
		{math.MaxUint64, true},
	} {
		t.Run(strconv.FormatUint(c.limit, 10), func(t *testing.T) {
			out, _, err := run(t, bytes.NewReader(data), "decode", "--max-payload", strconv.FormatUint(c.limit, 10))
			if c.ok {
				if err != nil || !strings.Contains(out, "aaaa") {
					t.Fatalf("err %v, output %q", err, out)
				}
				return
			}
			if got := ExitCode(err); got != ExitInvalidInput {
				t.Errorf("exit %d, want %d (%v)", got, ExitInvalidInput, err)
			}
			if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("larger than %d bytes (--max-payload)", c.limit)) {
				t.Errorf("error %v", err)
			}
		})
	}

	t.Run("0", func(t *testing.T) {
		_, _, err := run(t, bytes.NewReader(data), "decode", "--max-payload", "0")
		if got := ExitCode(err); got != ExitInvalidInput || !strings.Contains(err.Error(), "--max-payload must be at least 1") {
			t.Errorf("exit %d (%v)", got, err)
		}
	})

	t.Run("default", func(t *testing.T) {
		big := payloadOfSize(t, defaultMaxPayload)
		if _, _, err := run(t, bytes.NewReader(big), "decode", "--json"); err != nil {
			t.Fatalf("a payload of %d bytes: %v", len(big), err)
		}
		bigger := payloadOfSize(t, defaultMaxPayload+1)
		_, _, err := run(t, bytes.NewReader(bigger), "decode")
		if got := ExitCode(err); got != ExitInvalidInput {
			t.Errorf("a payload of %d bytes: exit %d, want %d (%v)", len(bigger), got, ExitInvalidInput, err)
		}
	})
}

// largeInput is an input of zero bytes, far longer than the bound, as a
// device or a pipe can be, that counts what was read of it.
type largeInput struct{ n, size int64 }

func (l *largeInput) Read(b []byte) (int, error) {
	if l.n >= l.size {
		return 0, io.EOF
	}
	b = b[:min(int64(len(b)), l.size-l.n)]
	clear(b)
	l.n += int64(len(b))
	return len(b), nil
}

// TestDecodeLargeInput reads an input far longer than the bound: decode
// stops after the bound and one byte, and allocates no more than a small
// multiple of the bound.
func TestDecodeLargeInput(t *testing.T) {
	for _, limit := range []int{1, 1 << 10, defaultMaxPayload} {
		t.Run(strconv.Itoa(limit), func(t *testing.T) {
			in := &largeInput{size: 4*int64(limit) + 1<<20}
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			_, _, err := run(t, in, "decode", "--max-payload", strconv.Itoa(limit))
			runtime.ReadMemStats(&after)
			if got := ExitCode(err); got != ExitInvalidInput {
				t.Errorf("exit %d, want %d (%v)", got, ExitInvalidInput, err)
			}
			if err == nil || !strings.Contains(err.Error(), "larger than "+strconv.Itoa(limit)+" bytes") {
				t.Errorf("error %v", err)
			}
			if in.n < int64(limit)+1 || in.n > int64(limit)+1 {
				t.Errorf("read %d bytes, want %d", in.n, limit+1)
			}
			// io.ReadAll grows its buffer step by step: up to about five
			// times what it returns, in all.
			if alloc := after.TotalAlloc - before.TotalAlloc; alloc > uint64(8*limit)+(1<<20) {
				t.Errorf("allocated %d bytes for a bound of %d", alloc, limit)
			}
		})
	}
}

func TestPayloadLimit(t *testing.T) {
	defer func(v uint64) { maxPayload = v }(maxPayload)
	maxPayload = 0
	if _, err := payloadLimit(); ExitCode(err) != ExitInvalidInput {
		t.Errorf("payloadLimit(0): %v", err)
	}
	// Above math.MaxInt - 1 (2^31 - 2 on a 32-bit platform) the bound is
	// math.MaxInt - 1: no input can be longer.
	for _, in := range []uint64{1, defaultMaxPayload - 1, defaultMaxPayload, defaultMaxPayload + 1,
		math.MaxInt32 - 1, math.MaxInt32, math.MaxInt32 + 1, math.MaxUint32, math.MaxUint32 + 1,
		math.MaxInt - 1, math.MaxInt, math.MaxInt64, math.MaxUint64} {
		maxPayload = in
		got, err := payloadLimit()
		if want := min(in, uint64(math.MaxInt-1)); err != nil || uint64(got) != want {
			t.Errorf("payloadLimit(%d) = %d, %v; want %d", in, got, err, want)
		}
	}
}

// TestListenMaxPayload sends listen a message above --max-payload and then
// one within it: the first is reported on stderr as not decoded, the second
// is printed, and listen runs on.
func TestListenMaxPayload(t *testing.T) {
	url := startBroker(t)
	small := encodeMetric(t, "within")
	for _, c := range []struct {
		name  string
		limit int
		args  []string
	}{
		{"flag", len(small), []string{"--max-payload", strconv.Itoa(len(small))}},
		{"default", defaultMaxPayload, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r := start(ctx, nil, append([]string{"listen", "--broker", url}, c.args...)...)
			eventually(t, "listening", contains(r.stderr, "Listening on spBv1.0/#"))

			pub := rawClient(t, url, "pub-"+c.name)
			over := payloadOfSize(t, c.limit+1)
			for _, m := range []struct {
				topic string
				data  []byte
			}{{"spBv1.0/g/NDATA/over", over}, {"spBv1.0/STATE/over", over}, {"spBv1.0/g/NDATA/n", small}} {
				if err := pub.Publish(context.Background(), m.topic, 1, false, m.data); err != nil {
					t.Fatal(err)
				}
			}
			eventually(t, "the message within the bound", contains(r.stdout, "within"))
			for _, topic := range []string{"spBv1.0/g/NDATA/over", "spBv1.0/STATE/over"} {
				want := fmt.Sprintf("decode error on %s: payload of %d bytes is above the limit of %d bytes (--max-payload)", topic, c.limit+1, c.limit)
				eventually(t, "the report of "+topic, contains(r.stderr, want))
			}
			if strings.Contains(r.stdout.String(), "over") {
				t.Errorf("the message above the bound was printed")
			}
			select {
			case err := <-r.done:
				t.Fatalf("listen ended: %v", err)
			default:
			}
			cancel()
			if err := r.wait(t); err != nil {
				t.Fatal(err)
			}
		})
	}
}
