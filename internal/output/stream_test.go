// SPDX-License-Identifier: MIT

package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"strings"
	"testing"
	"text/tabwriter"
	"time"

	"github.com/otfabric/go-sparkplugb/payload"
	pb "github.com/otfabric/go-sparkplugb/proto"
	"github.com/otfabric/go-sparkplugb/topic"
	"google.golang.org/protobuf/proto"
)

// wholeDocument is the JSON form as one json.Encoder call over the whole
// view, with a value JSON has no form for written as text: the form the
// metric-by-metric writer has to match byte for byte.
func wholeDocument(t *testing.T, view any) string {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(view); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func referenceMetrics(pl *pb.Payload) []MetricView {
	var out []MetricView
	for _, m := range pl.Metrics {
		mv := metricView(m)
		if _, err := json.Marshal(mv.Value); err != nil {
			mv.Value = fmt.Sprint(mv.Value)
		}
		out = append(out, mv)
	}
	return out
}

func jsonPayloads(t *testing.T) map[string]*pb.Payload {
	t.Helper()
	mk := func(name string, dt pb.DataType, v any) *pb.Payload_Metric {
		m, err := payload.NewMetric(name, dt, v)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	nan := mk("nan", pb.DataType_Double, math.NaN())
	arr := mk("arr", pb.DataType_DoubleArray, []float64{1, math.Inf(1), 3})
	far := mk("far", pb.DataType_DateTime, time.UnixMilli(1<<62))
	html := mk("<b>&", pb.DataType_String, "a<b>&\x1b\u202e")
	ints := mk("ints", pb.DataType_Int32Array, []int32{1, -2, 3})
	strs := mk("strs", pb.DataType_StringArray, []string{"a", "", "c"})
	return map[string]*pb.Payload{
		"sample":     samplePayload(),
		"empty":      {},
		"no metrics": {Seq: proto.Uint64(3), Timestamp: proto.Uint64(1_700_000_000_000)},
		"one":        payload.NewPayload(1, mk("a", pb.DataType_Int32, int32(1))),
		"not JSON":   payload.NewPayload(2, nan, arr, far),
		"strings":    payload.NewPayload(4, html, ints, strs),
	}
}

// TestPrintJSONSameDocument checks that writing the metrics one at a time
// gives the bytes of encoding the whole document at once.
func TestPrintJSONSameDocument(t *testing.T) {
	tp := topic.Topic{GroupID: "g", Type: topic.DDATA, EdgeNodeID: "n", DeviceID: "d"}
	for name, pl := range jsonPayloads(t) {
		t.Run(name, func(t *testing.T) {
			mv := MessageView{Topic: tp.String(), Type: tp.Type.String(), GroupID: "g", NodeID: "n", DeviceID: "d",
				Seq: pl.GetSeq(), Metrics: referenceMetrics(pl)}
			if pl.Timestamp != nil {
				mv.Time = millisTime(*pl.Timestamp)
			}
			var buf bytes.Buffer
			if err := (&Printer{W: &buf, JSON: true}).PrintMessage(tp, pl); err != nil {
				t.Fatal(err)
			}
			if want := wholeDocument(t, mv); buf.String() != want {
				t.Errorf("message:\n got %q\nwant %q", buf.String(), want)
			}

			pv := payloadView{Seq: pl.GetSeq(), Metrics: referenceMetrics(pl)}
			if pl.Timestamp != nil {
				pv.Time = millisTime(*pl.Timestamp)
			}
			buf.Reset()
			if err := (&Printer{W: &buf, JSON: true}).PrintPayload(pl); err != nil {
				t.Fatal(err)
			}
			if want := wholeDocument(t, pv); buf.String() != want {
				t.Errorf("payload:\n got %q\nwant %q", buf.String(), want)
			}
		})
	}
}

// TestPrintTextSameColumns checks that a message of up to textBlockLines
// metric lines is aligned as one block, as before the output was written in
// blocks.
func TestPrintTextSameColumns(t *testing.T) {
	pl := samplePayload()
	var want bytes.Buffer
	want.WriteString("seq=5  ts=2023-11-14T22:13:20Z\n")
	tw := tabwriter.NewWriter(&want, 2, 4, 2, ' ', 0)
	for _, m := range pl.Metrics {
		mv := metricView(m)
		line := fmt.Sprintf("  %s\t%s\t%v", mv.Name, mv.Type, mv.Value)
		if mv.Quality != "" {
			line += "\tquality=" + mv.Quality
		}
		_, _ = fmt.Fprintln(tw, line)
	}
	_ = tw.Flush()

	var got bytes.Buffer
	if err := (&Printer{W: &got}).PrintPayload(pl); err != nil {
		t.Fatal(err)
	}
	if got.String() != want.String() {
		t.Errorf("got\n%s\nwant\n%s", got.String(), want.String())
	}
}

// TestPrintTextBlocks checks a message of more metric lines than a block:
// every metric is printed, once, in order.
func TestPrintTextBlocks(t *testing.T) {
	n := textBlockLines*2 + 3
	ms := make([]*pb.Payload_Metric, n)
	for i := range ms {
		m, err := payload.NewMetric(fmt.Sprintf("m%d", i), pb.DataType_Int32, int32(i))
		if err != nil {
			t.Fatal(err)
		}
		ms[i] = m
	}
	var buf bytes.Buffer
	if err := (&Printer{W: &buf}).PrintPayload(&pb.Payload{Metrics: ms}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != n+1 {
		t.Fatalf("%d lines, want %d", len(lines), n+1)
	}
	for i, l := range lines[1:] {
		if f := strings.Fields(l); len(f) != 3 || f[0] != fmt.Sprintf("m%d", i) || f[2] != fmt.Sprint(i) {
			t.Fatalf("line %d = %q", i+1, l)
		}
	}
}

// heapSampler is a writer that discards what it is given and records the
// largest heap in use after a collection, every samplerEvery writes and at
// every write of at least samplerLarge bytes.
type heapSampler struct {
	writes int
	peak   uint64
}

const (
	samplerEvery = 512
	samplerLarge = 64 << 10
)

func (h *heapSampler) Write(b []byte) (int, error) {
	h.writes++
	if h.writes%samplerEvery == 0 || len(b) >= samplerLarge {
		h.sample()
	}
	return len(b), nil
}

func (h *heapSampler) sample() {
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	h.peak = max(h.peak, ms.HeapAlloc)
}

func heapInUse() uint64 {
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ms.HeapAlloc
}

// outputBudget bounds what printing one payload may hold on the heap at once,
// beyond the decoded payload itself: one block of text lines or one metric's
// JSON, not the whole output.
const outputBudget = 4 << 20

// boolArrays encodes a payload of n BooleanArray metrics of 64 elements
// each, without names or timestamps.
func boolArrays(t *testing.T, n int) []byte {
	t.Helper()
	bools := make([]bool, 64)
	ms := make([]*pb.Payload_Metric, n)
	for i := range ms {
		m, err := payload.NewMetric("", pb.DataType_BooleanArray, bools)
		if err != nil {
			t.Fatal(err)
		}
		m.Timestamp = nil
		ms[i] = m
	}
	wire, err := payload.Encode(&pb.Payload{Metrics: ms})
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

// TestPrintMemoryBudget prints a payload of many array metrics, whose output
// is about 50 times its size on the wire in text and 150 times in JSON, and
// checks that the heap in use while it is printed stays within outputBudget:
// the output is written as it is formatted, not held until the end.
func TestPrintMemoryBudget(t *testing.T) {
	wire := boolArrays(t, 1<<14)
	pl, err := payload.Decode(wire)
	if err != nil {
		t.Fatal(err)
	}
	tp := topic.Topic{GroupID: "g", Type: topic.NDATA, EdgeNodeID: "n"}
	for _, c := range []struct {
		name  string
		json  bool
		print func(*Printer) error
	}{
		{"message text", false, func(p *Printer) error { return p.PrintMessage(tp, pl) }},
		{"message JSON", true, func(p *Printer) error { return p.PrintMessage(tp, pl) }},
		{"payload text", false, func(p *Printer) error { return p.PrintPayload(pl) }},
		{"payload JSON", true, func(p *Printer) error { return p.PrintPayload(pl) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			base := heapInUse()
			h := &heapSampler{}
			if err := c.print(&Printer{W: h, JSON: c.json}); err != nil {
				t.Fatal(err)
			}
			h.sample()
			var held uint64
			if h.peak > base {
				held = h.peak - base
			}
			t.Logf("wire %d B, %d writes, heap held while printing %d B", len(wire), h.writes, held)
			if held > outputBudget {
				t.Errorf("printing held %d bytes on the heap, budget %d (wire size %d)", held, outputBudget, len(wire))
			}
		})
	}
	runtime.KeepAlive(pl)
}

// failWriter accepts limit bytes, then fails every write.
type failWriter struct {
	limit, n int
	writes   int
}

var errFull = errors.New("no space left")

func (f *failWriter) Write(b []byte) (int, error) {
	f.writes++
	if f.n+len(b) > f.limit {
		return 0, errFull
	}
	f.n += len(b)
	return len(b), nil
}

// TestPrintWriteError checks that each Print method returns a failed write
// as a *WriteError, from the first write or a later one, and stops writing.
func TestPrintWriteError(t *testing.T) {
	ms := make([]*pb.Payload_Metric, 100)
	for i := range ms {
		ms[i], _ = payload.NewMetric("m", pb.DataType_Int32, int32(i))
	}
	pl := payload.NewPayload(1, ms...)
	tp := topic.Topic{GroupID: "g", Type: topic.NDATA, EdgeNodeID: "n"}
	st, _ := topic.Parse("spBv1.0/STATE/h")
	state := []byte(`{"online":true,"timestamp":1}`)
	for _, js := range []bool{false, true} {
		for name, print := range map[string]func(*Printer) error{
			"message": func(p *Printer) error { return p.PrintMessage(tp, pl) },
			"payload": func(p *Printer) error { return p.PrintPayload(pl) },
			"state":   func(p *Printer) error { return p.PrintState(st, state) },
		} {
			for _, limit := range []int{0, 10} {
				t.Run(fmt.Sprintf("%s/json=%t/after %d bytes", name, js, limit), func(t *testing.T) {
					w := &failWriter{limit: limit}
					err := print(&Printer{W: w, JSON: js})
					var we *WriteError
					if !errors.As(err, &we) || !errors.Is(err, errFull) {
						t.Fatalf("err = %v, want a *WriteError wrapping the write error", err)
					}
					if w.writes > 3 {
						t.Errorf("%d writes after the failure, want the printer to stop", w.writes-1)
					}
				})
			}
		}
	}
	if err := (&Printer{W: io.Discard}).PrintState(st, []byte("x")); err == nil || errors.As(err, new(*WriteError)) {
		t.Errorf("a STATE payload that does not decode: err = %v, want a decode error", err)
	}
}
