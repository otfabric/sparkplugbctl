// SPDX-License-Identifier: MIT

package output

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/otfabric/go-sparkplugb/payload"
	pb "github.com/otfabric/go-sparkplugb/proto"
	"github.com/otfabric/go-sparkplugb/topic"
	"google.golang.org/protobuf/proto"
)

func samplePayload() *pb.Payload {
	good, _ := payload.NewMetric("good", pb.DataType_Double, 1.5)
	payload.SetQuality(good, payload.QualityGood)
	good.Timestamp = proto.Uint64(1_700_000_000_000)

	bad, _ := payload.NewMetric("bad", pb.DataType_Int32, int32(2))
	payload.SetQuality(bad, payload.QualityBad)

	stale, _ := payload.NewMetric("stale", pb.DataType_Boolean, true)
	payload.SetQuality(stale, payload.QualityStale)

	odd, _ := payload.NewMetric("odd", pb.DataType_String, "x")
	payload.SetQuality(odd, 42)

	aliased := &pb.Payload_Metric{
		Alias:    proto.Uint64(9),
		Datatype: proto.Uint32(uint32(pb.DataType_Int32)),
		Value:    &pb.Payload_Metric_IntValue{IntValue: 7},
	}
	novalue := &pb.Payload_Metric{Name: proto.String("empty"), Datatype: proto.Uint32(uint32(pb.DataType_Int32))}

	p := payload.NewPayload(5, good, bad, stale, odd, aliased, novalue)
	p.Timestamp = proto.Uint64(1_700_000_000_000)
	return p
}

func TestPrintMessageText(t *testing.T) {
	var buf bytes.Buffer
	tp := topic.Topic{GroupID: "g", Type: topic.DDATA, EdgeNodeID: "n", DeviceID: "d"}
	(&Printer{W: &buf}).PrintMessage(tp, samplePayload())
	out := buf.String()
	for _, want := range []string{
		"[DDATA] spBv1.0/g/DDATA/n/d", "seq=5", "ts=2023-11-14T22:13:20Z",
		"good", "Double", "quality=GOOD", "quality=BAD", "quality=STALE", "quality=42", "alias/9", "empty",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	buf.Reset()
	(&Printer{W: &buf}).PrintMessage(tp, &pb.Payload{})
	if strings.Contains(buf.String(), "ts=") {
		t.Errorf("unexpected timestamp:\n%s", buf.String())
	}
}

func TestPrintMessageJSON(t *testing.T) {
	var buf bytes.Buffer
	tp := topic.Topic{GroupID: "g", Type: topic.NBIRTH, EdgeNodeID: "n"}
	(&Printer{W: &buf, JSON: true}).PrintMessage(tp, samplePayload())
	var mv MessageView
	if err := json.Unmarshal(buf.Bytes(), &mv); err != nil {
		t.Fatal(err)
	}
	if mv.Type != "NBIRTH" || mv.GroupID != "g" || mv.NodeID != "n" || mv.Seq != 5 || len(mv.Metrics) != 6 {
		t.Fatalf("view = %+v", mv)
	}
	if mv.Metrics[0].Timestamp == "" || mv.Metrics[4].Alias != 9 || mv.Metrics[4].Name != "alias/9" {
		t.Fatalf("metrics = %+v", mv.Metrics)
	}
}

func TestPrintState(t *testing.T) {
	tp := topic.Topic{Type: topic.STATE, EdgeNodeID: "scada"}
	var buf bytes.Buffer
	if err := (&Printer{W: &buf}).PrintState(tp, []byte(`{"online":false,"timestamp":1700000000000}`)); err != nil {
		t.Fatal(err)
	}
	if want := "[STATE] spBv1.0/STATE/scada\n  online=false  ts=2023-11-14T22:13:20Z\n\n"; buf.String() != want {
		t.Errorf("text = %q, want %q", buf.String(), want)
	}

	buf.Reset()
	if err := (&Printer{W: &buf}).PrintState(tp, []byte(`{"online":true}`)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "ts=") {
		t.Errorf("unexpected timestamp:\n%s", buf.String())
	}

	buf.Reset()
	if err := (&Printer{W: &buf, JSON: true}).PrintState(tp, []byte(`{"online":true,"timestamp":1700000000000}`)); err != nil {
		t.Fatal(err)
	}
	var sv StateView
	if err := json.Unmarshal(buf.Bytes(), &sv); err != nil {
		t.Fatal(err)
	}
	if sv != (StateView{Topic: "spBv1.0/STATE/scada", Type: "STATE", HostID: "scada", Online: true, Time: "2023-11-14T22:13:20Z"}) {
		t.Errorf("view = %+v", sv)
	}

	for _, bad := range []string{`not json`, `{"timestamp":1}`} {
		if err := (&Printer{W: &buf}).PrintState(tp, []byte(bad)); err == nil {
			t.Errorf("PrintState(%s): want error", bad)
		}
	}
}

func TestPrintPayload(t *testing.T) {
	var buf bytes.Buffer
	(&Printer{W: &buf}).PrintPayload(samplePayload())
	out := buf.String()
	for _, want := range []string{"seq=5", "ts=2023-11-14T22:13:20Z", "quality=GOOD", "alias/9"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	buf.Reset()
	(&Printer{W: &buf}).PrintPayload(&pb.Payload{})
	if strings.Contains(buf.String(), "ts=") {
		t.Errorf("unexpected timestamp: %s", buf.String())
	}

	buf.Reset()
	(&Printer{W: &buf, JSON: true}).PrintPayload(samplePayload())
	var v struct {
		Seq     uint64       `json:"seq"`
		Time    string       `json:"timestamp"`
		Metrics []MetricView `json:"metrics"`
	}
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	if v.Seq != 5 || v.Time == "" || len(v.Metrics) != 6 {
		t.Fatalf("json payload = %+v", v)
	}

	buf.Reset()
	(&Printer{W: &buf, JSON: true}).PrintPayload(&pb.Payload{})
	if strings.Contains(buf.String(), "timestamp") {
		t.Errorf("empty timestamp not omitted: %s", buf.String())
	}
}

// A metric reading NaN or infinity used to make the JSON encoder refuse the
// whole document, and --json printed nothing at all.
func TestPrintJSONNonFinite(t *testing.T) {
	var metrics []*pb.Payload_Metric
	for name, v := range map[string]float64{"nan": math.NaN(), "up": math.Inf(1), "down": math.Inf(-1)} {
		m, err := payload.NewMetric(name, pb.DataType_Double, v)
		if err != nil {
			t.Fatal(err)
		}
		metrics = append(metrics, m)
	}
	f, err := payload.NewMetric("f", pb.DataType_Float, float32(math.NaN()))
	if err != nil {
		t.Fatal(err)
	}
	ok, err := payload.NewMetric("ok", pb.DataType_Double, 1.5)
	if err != nil {
		t.Fatal(err)
	}
	far, err := payload.NewMetric("far", pb.DataType_DateTime, time.UnixMilli(1<<62))
	if err != nil {
		t.Fatal(err)
	}
	pl := payload.NewPayload(1, append(metrics, f, ok, far)...)

	want := map[string]any{"nan": "NaN", "up": "+Inf", "down": "-Inf", "f": "NaN", "ok": 1.5}
	var buf bytes.Buffer
	tp := topic.Topic{GroupID: "g", Type: topic.NDATA, EdgeNodeID: "n"}
	for name, print := range map[string]func(*Printer){
		"payload": func(p *Printer) { p.PrintPayload(pl) },
		"message": func(p *Printer) { p.PrintMessage(tp, pl) },
	} {
		buf.Reset()
		print(&Printer{W: &buf, JSON: true})
		var doc struct {
			Metrics []MetricView `json:"metrics"`
		}
		if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
			t.Fatalf("%s: %v: %q", name, err, buf.String())
		}
		if len(doc.Metrics) != len(want)+1 {
			t.Fatalf("%s: metrics = %+v", name, doc.Metrics)
		}
		for _, m := range doc.Metrics {
			if m.Name == "far" {
				if _, ok := m.Value.(string); !ok {
					t.Errorf("%s: far = %v", name, m.Value)
				}
				continue
			}
			if m.Value != want[m.Name] {
				t.Errorf("%s: %s = %v, want %v", name, m.Name, m.Value, want[m.Name])
			}
		}
	}
}
