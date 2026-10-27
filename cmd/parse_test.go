// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/otfabric/go-sparkplugb/payload"
	pb "github.com/otfabric/go-sparkplugb/proto"
)

func TestParseMetricSpecAllTypes(t *testing.T) {
	cases := []struct {
		spec string
		dt   pb.DataType
		want any
	}{
		{"a:Int8=-8", pb.DataType_Int8, int8(-8)},
		{"a:int16=-16", pb.DataType_Int16, int16(-16)},
		{"a:Int32=-32", pb.DataType_Int32, int32(-32)},
		{"a:Int64=-64", pb.DataType_Int64, int64(-64)},
		{"a:UInt8=8", pb.DataType_UInt8, uint8(8)},
		{"a:UInt16=16", pb.DataType_UInt16, uint16(16)},
		{"a:UInt32=32", pb.DataType_UInt32, uint32(32)},
		{"a:UInt64=64", pb.DataType_UInt64, uint64(64)},
		{"a:Float=1.5", pb.DataType_Float, float32(1.5)},
		{"a:Double=2.5", pb.DataType_Double, 2.5},
		{"a:Boolean=true", pb.DataType_Boolean, true},
		{"a:bool=false", pb.DataType_Boolean, false},
		{"a:String=x=y", pb.DataType_String, "x=y"},
		{"a:Text=hello", pb.DataType_Text, "hello"},
		{"a:UUID=123e4567-e89b-12d3-a456-426614174000", pb.DataType_UUID, "123e4567-e89b-12d3-a456-426614174000"},
	}
	for _, c := range cases {
		t.Run(c.spec, func(t *testing.T) {
			m, err := parseMetricSpec(c.spec)
			if err != nil {
				t.Fatal(err)
			}
			if m.GetName() != "a" || pb.DataType(m.GetDatatype()) != c.dt {
				t.Fatalf("got name=%q type=%v", m.GetName(), pb.DataType(m.GetDatatype()))
			}
			v, err := payload.GetValue(m)
			if err != nil {
				t.Fatal(err)
			}
			if v != c.want {
				t.Fatalf("value = %#v, want %#v", v, c.want)
			}
		})
	}

	m, err := parseMetricSpec("blob:Bytes=abc")
	if err != nil {
		t.Fatal(err)
	}
	v, _ := payload.GetValue(m)
	if !bytes.Equal(v.([]byte), []byte("abc")) {
		t.Fatalf("bytes value = %v", v)
	}
}

func TestParseMetricSpecErrors(t *testing.T) {
	for _, spec := range []string{
		"noColon",
		"name:Double",
		"name:Nope=1",
		"name:Int8=999",
		"name:Int16=x",
		"name:Int32=x",
		"name:Int64=x",
		"name:UInt8=-1",
		"name:UInt16=x",
		"name:UInt32=x",
		"name:UInt64=x",
		"name:Float=x",
		"name:Double=x",
		"name:Boolean=maybe",
	} {
		if _, err := parseMetricSpec(spec); err == nil {
			t.Errorf("parseMetricSpec(%q) succeeded, want error", spec)
		}
	}
}

func TestParseValueUnsupported(t *testing.T) {
	if _, err := parseValue(pb.DataType_DateTime, "1"); err == nil {
		t.Fatal("want error for unsupported type")
	}
}

func TestParseMetrics(t *testing.T) {
	ms, err := parseMetrics([]string{"a:Int32=1", "b:String=x"})
	if err != nil || len(ms) != 2 {
		t.Fatalf("parseMetrics = %v, %v", ms, err)
	}
	if _, err := parseMetrics([]string{"a:Int32=1", "bad"}); err == nil || !strings.Contains(err.Error(), `"bad"`) {
		t.Fatalf("want error naming the spec, got %v", err)
	}
	if ms, err := parseMetrics(nil); err != nil || ms != nil {
		t.Fatalf("parseMetrics(nil) = %v, %v", ms, err)
	}
}

func TestBuildEdgeMetrics(t *testing.T) {
	ms, err := buildEdgeMetrics(nil)
	if err != nil || len(ms) != 3 {
		t.Fatalf("defaults = %d metrics, %v", len(ms), err)
	}
	ms, err = buildEdgeMetrics([]string{"x:Int32=1"})
	if err != nil || len(ms) != 1 || ms[0].GetName() != "x" {
		t.Fatalf("custom = %v, %v", ms, err)
	}
	if _, err := buildEdgeMetrics([]string{"bad"}); err == nil {
		t.Fatal("want error")
	}
}

func TestCLILogger(t *testing.T) {
	var buf bytes.Buffer
	l := &cliLogger{w: &buf}
	l.Debugf("d %d", 1)
	l.Infof("i %d", 2)
	l.Warnf("w %d", 3)
	l.Errorf("e %d", 4)
	for _, want := range []string{"[DEBUG] d 1\n", "[INFO]  i 2\n", "[WARN]  w 3\n", "[ERROR] e 4\n"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("log output missing %q:\n%s", want, buf.String())
		}
	}
}

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	m, _ := payload.NewMetric("a", pb.DataType_Int32, int32(1))
	printJSON(&buf, "DBIRTH", "g", "n", "d", []*pb.Payload_Metric{m})
	out := buf.String()
	for _, want := range []string{`"type":"DBIRTH"`, `"group_id":"g"`, `"node_id":"n"`, `"device_id":"d"`, `"metrics":1`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in %s", want, out)
		}
	}
	buf.Reset()
	printJSON(&buf, "NDEATH", "g", "n", "", nil)
	if strings.Contains(buf.String(), "device_id") || strings.Contains(buf.String(), `"metrics"`) {
		t.Errorf("empty fields not omitted: %s", buf.String())
	}
}
