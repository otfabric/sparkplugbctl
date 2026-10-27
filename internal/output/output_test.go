// SPDX-License-Identifier: MIT

package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

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
