// SPDX-License-Identifier: MIT

package output

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/otfabric/go-sparkplugb/payload"
	pb "github.com/otfabric/go-sparkplugb/proto"
	"github.com/otfabric/go-sparkplugb/topic"
	"google.golang.org/protobuf/proto"
)

func seedPayloads(f *testing.F) {
	f.Helper()
	add := func(name string, dt pb.DataType, v any) {
		m, err := payload.NewMetric(name, dt, v)
		if err != nil {
			f.Fatal(err)
		}
		b, err := proto.Marshal(payload.NewPayload(1, m))
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	add("t", pb.DataType_Double, 22.5)
	add("f", pb.DataType_Float, float32(1.5))
	add("nan", pb.DataType_Double, math.NaN())
	add("inf", pb.DataType_Float, float32(math.Inf(1)))
	add("b", pb.DataType_Boolean, true)
	add("s", pb.DataType_String, "x")
	add("i", pb.DataType_Int32, int32(-1))
	add("u", pb.DataType_UInt64, uint64(1<<63))
	add("d", pb.DataType_Bytes, []byte{0, 1, 2})
	add("far", pb.DataType_DateTime, time.UnixMilli(1<<62))
	f.Add([]byte{})
	f.Add([]byte{0xff, 0xff})
}

// FuzzPrintPayload feeds what arrives from a broker or a file. The text form
// must not panic, and the JSON form must always be one JSON document: scripts
// read it.
func FuzzPrintPayload(f *testing.F) {
	seedPayloads(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		pl, err := payload.Decode(data)
		if err != nil {
			return
		}
		(&Printer{W: &bytes.Buffer{}}).PrintPayload(pl)
		var out bytes.Buffer
		(&Printer{W: &out, JSON: true}).PrintPayload(pl)
		if !json.Valid(out.Bytes()) {
			t.Fatalf("not one JSON document: %q", out.Bytes())
		}
		out.Reset()
		tp, err := topic.Parse("spBv1.0/g/NDATA/n")
		if err != nil {
			t.Fatal(err)
		}
		(&Printer{W: &out, JSON: true}).PrintMessage(tp, pl)
		if !json.Valid(out.Bytes()) {
			t.Fatalf("message is not one JSON document: %q", out.Bytes())
		}
	})
}

// FuzzPrintState feeds STATE messages, JSON from whoever can publish to the
// broker.
func FuzzPrintState(f *testing.F) {
	f.Add([]byte(`{"online":true,"timestamp":1}`))
	f.Add([]byte(`{"online":false}`))
	f.Add([]byte(`not json`))
	f.Add([]byte(`{"timestamp":-1}`))
	tp, err := topic.Parse("spBv1.0/STATE/host")
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		var out bytes.Buffer
		if err := (&Printer{W: &out, JSON: true}).PrintState(tp, data); err != nil {
			return
		}
		if !json.Valid(out.Bytes()) {
			t.Fatalf("not one JSON document: %q", out.Bytes())
		}
		if err := (&Printer{W: &bytes.Buffer{}}).PrintState(tp, data); err != nil {
			t.Fatalf("JSON form accepted, text form refused: %v", err)
		}
	})
}
