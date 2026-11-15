// SPDX-License-Identifier: MIT

package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/otfabric/go-sparkplugb/payload"
	pb "github.com/otfabric/go-sparkplugb/proto"
	"github.com/otfabric/go-sparkplugb/topic"
)

func TestEscape(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"", ""},
		{"temperature", "temperature"},
		{"Node Control/Rebirth", "Node Control/Rebirth"},
		{"Température °C — 温度", "Température °C — 温度"},
		{`back\slash "quoted"`, `back\slash "quoted"`},
		{"a\x1b[2Jb", `a\x1b[2Jb`},
		{"bell\a", `bell\a`},
		{"tab\tnewline\ncr\r", `tab\tnewline\ncr\r`},
		{"nul\x00", `nul\x00`},
		{"del\x7f", `del\x7f`},
		{"c1\u0085\u009b", `c1\u0085\u009b`},
		{"rlo\u202eolleh", `rlo\u202eolleh`},
		{"\u061c\u200e\u200f\u202a\u202b\u202c\u202d\u2066\u2067\u2068\u2069", `\u061c\u200e\u200f\u202a\u202b\u202c\u202d\u2066\u2067\u2068\u2069`},
		{"line\u2028para\u2029", `line\u2028para\u2029`},
		{"bad\xffutf8\xc3", `bad\xffutf8\xc3`},
		{"\ufffd kept", "\ufffd kept"},
		{"zero\u200bwidth", "zero\u200bwidth"},
	} {
		got := Escape(c.in)
		if got != c.want {
			t.Errorf("Escape(%q) = %q, want %q", c.in, got, c.want)
		}
		if !utf8.ValidString(got) {
			t.Errorf("Escape(%q) = %q is not valid UTF-8", c.in, got)
		}
		for _, r := range got {
			if escaped(r) {
				t.Errorf("Escape(%q) = %q keeps %U", c.in, got, r)
			}
		}
	}
}

// rawControls are what text output must not contain: what a terminal acts on.
func rawControls(s string) []rune {
	var out []rune
	for _, r := range s {
		if r != '\n' && r != '\t' && (escaped(r) || r == utf8.RuneError) {
			out = append(out, r)
		}
	}
	return out
}

// TestPrintTextEscapes prints a message whose topic IDs, metric names and
// string values carry control characters, bidirectional formatting
// characters and invalid UTF-8: the text form has none of them raw; the JSON
// form still has the strings as they arrived.
func TestPrintTextEscapes(t *testing.T) {
	controls := "x\x1b]0;title\a\x1b[2J\u202e\u009b\xff"
	tp, err := topic.Parse("spBv1.0/g" + "\x1b[31m" + "/DDATA/n\u202e/d\u0085")
	if err != nil {
		t.Fatal(err)
	}
	s, _ := payload.NewMetric("name"+controls, pb.DataType_String, "value"+controls)
	tx, _ := payload.NewMetric("text", pb.DataType_Text, "line1\nline2\ttab")
	arr, _ := payload.NewMetric("arr", pb.DataType_StringArray, []string{"a", controls})
	plain, _ := payload.NewMetric("plain", pb.DataType_String, "ok")
	pl := payload.NewPayload(1, s, tx, arr, plain)

	var buf bytes.Buffer
	_ = (&Printer{W: &buf}).PrintMessage(tp, pl)
	_ = (&Printer{W: &buf}).PrintPayload(pl)
	st, _ := topic.Parse("spBv1.0/STATE/h\x1b\u202e")
	_ = (&Printer{W: &buf}).PrintState(st, []byte(`{"online":true}`))
	out := buf.String()
	if raw := rawControls(out); len(raw) > 0 {
		t.Errorf("text output has raw %q:\n%s", string(raw), out)
	}
	esc := `x\x1b]0;title\a\x1b[2J\u202e\u009b\xff`
	for _, want := range []string{`g\x1b[31m/DDATA/n\u202e/d\u0085`, "name" + esc, "value" + esc, "[a " + esc + "]",
		`line1\nline2\ttab`, `spBv1.0/STATE/h\x1b\u202e`, "plain", " ok\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output lacks %q:\n%s", want, out)
		}
	}
	if strings.Count(out, "\n") != 2+4+1+1+4+3 {
		t.Errorf("a value's newline started a line of its own:\n%s", out)
	}

	buf.Reset()
	_ = (&Printer{W: &buf, JSON: true}).PrintMessage(tp, pl)
	var mv MessageView
	if err := json.Unmarshal(buf.Bytes(), &mv); err != nil {
		t.Fatal(err)
	}
	if mv.GroupID != "g\x1b[31m" || mv.Metrics[1].Value != "line1\nline2\ttab" {
		t.Errorf("JSON changed the strings: %+v", mv)
	}
}
