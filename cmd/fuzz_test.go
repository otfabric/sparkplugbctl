// SPDX-License-Identifier: MIT

package cmd

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/otfabric/go-sparkplugb/payload"
)

// FuzzParseMetricSpec feeds --metric arguments. What parses must encode, and
// decode to the same name, type and value.
func FuzzParseMetricSpec(f *testing.F) {
	for _, s := range []string{
		"temperature:Double=22.5", "status:Boolean=true", "n:Int8=-128", "n:UInt64=18446744073709551615",
		"s:String=a=b:c", "u:UUID=x", "f:Float=NaN", "b:Bytes=00ff", "no separator", "n:Nope=1", ":String=", "x:bool=",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, spec string) {
		m, err := parseMetricSpec(spec)
		if err != nil {
			return
		}
		data, err := encodePayload(payload.NewPayload(0, m))
		if err != nil {
			if strings.ToValidUTF8(spec, "") != spec {
				return // Protobuf strings are UTF-8: the error is the answer
			}
			t.Fatalf("parsed but does not encode: %q: %v", spec, err)
		}
		back, err := payload.Decode(data)
		if err != nil {
			t.Fatalf("does not decode: %q: %v", spec, err)
		}
		if len(back.Metrics) != 1 || back.Metrics[0].GetName() != m.GetName() || back.Metrics[0].GetDatatype() != m.GetDatatype() {
			t.Fatalf("changed: %q: %v", spec, back)
		}
		want, err := payload.GetValue(m)
		if err != nil {
			t.Fatal(err)
		}
		got, err := payload.GetValue(back.Metrics[0])
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) && !nan(got) {
			t.Fatalf("value of %q: %#v became %#v", spec, want, got)
		}
	})
}

func nan(v any) bool {
	switch x := v.(type) {
	case float32:
		return math.IsNaN(float64(x))
	case float64:
		return math.IsNaN(x)
	}
	return false
}
