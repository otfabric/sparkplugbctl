// SPDX-License-Identifier: MIT

package output

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/otfabric/go-sparkplugb/payload"
	pb "github.com/otfabric/go-sparkplugb/proto"
	"github.com/otfabric/go-sparkplugb/topic"
)

// Printer writes decoded payloads to W, as text or, with JSON, as indented
// JSON documents (docs/json-output.md). Each metric is formatted and written
// before the next: what a payload holds in memory while it is printed does
// not grow with its number of metrics.
type Printer struct {
	W    io.Writer
	JSON bool
}

// WriteError is a write to the Printer's writer that failed. The Print
// methods return it, and stop at the first failed write.
type WriteError struct{ Err error }

func (e *WriteError) Error() string { return "write output: " + e.Err.Error() }
func (e *WriteError) Unwrap() error { return e.Err }

// errWriter passes writes to w until one fails, and then keeps that error.
type errWriter struct {
	w   io.Writer
	err error
}

func (e *errWriter) Write(b []byte) (int, error) {
	if e.err != nil {
		return 0, e.err
	}
	n, err := e.w.Write(b)
	if err != nil {
		e.err = err
	}
	return n, err
}

func (e *errWriter) WriteString(s string) {
	_, _ = io.WriteString(e, s)
}

func (e *errWriter) result() error {
	if e.err != nil {
		return &WriteError{Err: e.err}
	}
	return nil
}

type MessageView struct {
	Topic    string       `json:"topic"`
	Type     string       `json:"type"`
	GroupID  string       `json:"group_id,omitempty"`
	NodeID   string       `json:"node_id,omitempty"`
	DeviceID string       `json:"device_id,omitempty"`
	Seq      uint64       `json:"seq"`
	Time     string       `json:"timestamp"`
	Metrics  []MetricView `json:"metrics"`
}

// payloadView is the document of PrintPayload.
type payloadView struct {
	Seq     uint64       `json:"seq"`
	Time    string       `json:"timestamp,omitempty"`
	Metrics []MetricView `json:"metrics"`
}

// StateView is a primary host's STATE message: JSON, not protobuf.
type StateView struct {
	Topic  string `json:"topic"`
	Type   string `json:"type"`
	HostID string `json:"host_id"`
	Online bool   `json:"online"`
	Time   string `json:"timestamp"`
}

// PrintState prints the STATE message data of topic t (the JSON payload
// {"online", "timestamp"} of Sparkplug 3.0). Data that is not such a payload
// returns an error and prints nothing; a failed write returns a *WriteError.
func (p *Printer) PrintState(t topic.Topic, data []byte) error {
	var st struct {
		Online    *bool  `json:"online"`
		Timestamp uint64 `json:"timestamp"`
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return fmt.Errorf("STATE payload: %w", err)
	}
	if st.Online == nil {
		return fmt.Errorf("STATE payload: no \"online\"")
	}
	sv := StateView{Topic: t.String(), Type: t.Type.String(), HostID: t.EdgeNodeID, Online: *st.Online}
	if st.Timestamp != 0 {
		sv.Time = millisTime(st.Timestamp)
	}
	w := &errWriter{w: p.W}
	if p.JSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(sv) // a StateView always encodes; a write error is kept in w
		return w.result()
	}
	w.WriteString("[" + t.Type.String() + "] " + Escape(t.String()) + "\n")
	w.WriteString("  online=" + strconv.FormatBool(sv.Online))
	if sv.Time != "" {
		w.WriteString("  ts=" + sv.Time)
	}
	w.WriteString("\n\n")
	return w.result()
}

type MetricView struct {
	Name      string `json:"name"`
	Alias     uint64 `json:"alias,omitempty"`
	Type      string `json:"type"`
	Value     any    `json:"value"`
	Timestamp string `json:"timestamp,omitempty"`
	Quality   string `json:"quality,omitempty"`
}

// PrintMessage prints a message that arrived on topic t. A failed write
// returns a *WriteError.
func (p *Printer) PrintMessage(t topic.Topic, pl *pb.Payload) error {
	w := &errWriter{w: p.W}
	mv := MessageView{
		Topic:    t.String(),
		Type:     t.Type.String(),
		GroupID:  t.GroupID,
		NodeID:   t.EdgeNodeID,
		DeviceID: t.DeviceID,
		Seq:      pl.GetSeq(),
	}
	if pl.Timestamp != nil {
		mv.Time = millisTime(*pl.Timestamp)
	}

	if p.JSON {
		writeJSON(w, mv, pl.Metrics)
		return w.result()
	}

	w.WriteString("[" + t.Type.String() + "] " + Escape(t.String()) + "\n")
	w.WriteString("  seq=" + strconv.FormatUint(mv.Seq, 10))
	if mv.Time != "" {
		w.WriteString("  ts=" + mv.Time)
	}
	w.WriteString("\n")
	writeMetricLines(w, pl.Metrics)
	w.WriteString("\n")
	return w.result()
}

// PrintPayload prints a payload read from a file or stdin. A failed write
// returns a *WriteError.
func (p *Printer) PrintPayload(pl *pb.Payload) error {
	w := &errWriter{w: p.W}
	if p.JSON {
		v := payloadView{Seq: pl.GetSeq()}
		if pl.Timestamp != nil {
			v.Time = millisTime(*pl.Timestamp)
		}
		writeJSON(w, v, pl.Metrics)
		return w.result()
	}

	w.WriteString("seq=" + strconv.FormatUint(pl.GetSeq(), 10))
	if pl.Timestamp != nil {
		w.WriteString("  ts=" + millisTime(*pl.Timestamp))
	}
	w.WriteString("\n")
	writeMetricLines(w, pl.Metrics)
	return w.result()
}

// Text output aligns the metric lines of a message in columns
// (text/tabwriter), which holds the lines until it is flushed. It is flushed
// after textBlockLines lines, or once the lines it holds reach
// textBlockBytes, so that a message of many or very long lines is printed
// in blocks, each aligned on its own.
const (
	textBlockLines = 4096
	textBlockBytes = 256 << 10
)

// writeMetricLines writes one text line per metric: name, datatype, value
// and quality, aligned in columns.
func writeMetricLines(w *errWriter, metrics []*pb.Payload_Metric) {
	tw := tabwriter.NewWriter(w, 2, 4, 2, ' ', 0)
	lines, held := 0, 0
	for _, m := range metrics {
		mv := metricView(m)
		cells := []string{"  ", Escape(mv.Name), "\t", mv.Type, "\t", Escape(fmt.Sprint(mv.Value))}
		if mv.Quality != "" {
			cells = append(cells, "\tquality=", mv.Quality)
		}
		cells = append(cells, "\n")
		for _, c := range cells {
			_, _ = io.WriteString(tw, c)
			held += len(c)
		}
		lines++
		if lines == textBlockLines || held >= textBlockBytes {
			_ = tw.Flush()
			lines, held = 0, 0
		}
		if w.err != nil {
			return
		}
	}
	_ = tw.Flush()
}

// writeJSON writes head, a view whose last field is its nil "metrics", as
// json.Encoder with SetIndent("", "  ") writes it, with the metrics in place
// of the null: the same bytes as encoding the whole view at once, without
// holding the whole document. head always encodes (strings and integers).
func writeJSON(w *errWriter, head any, metrics []*pb.Payload_Metric) {
	b, err := json.MarshalIndent(head, "", "  ")
	if err != nil {
		w.err = err
		return
	}
	if len(metrics) == 0 {
		_, _ = w.Write(append(b, '\n'))
		return
	}
	const null = "null\n}"
	_, _ = w.Write(b[:len(b)-len(null)])
	w.WriteString("[\n")
	for i, m := range metrics {
		w.WriteString("    ")
		_, _ = w.Write(metricJSON(metricView(m)))
		if i < len(metrics)-1 {
			w.WriteString(",\n")
		} else {
			w.WriteString("\n")
		}
		if w.err != nil {
			return
		}
	}
	w.WriteString("  ]\n}\n")
}

// metricJSON encodes mv as an element of the "metrics" array. A value JSON
// has no form for, NaN and the infinities alone or in an array and a
// DateTime beyond year 9999, is written as text instead: left in, it would
// make the document unreadable, and one metric reading NaN (a sensor fault,
// say) would lose the others.
func metricJSON(mv MetricView) []byte {
	b, err := json.MarshalIndent(mv, "    ", "  ")
	if err != nil {
		mv.Value = fmt.Sprint(mv.Value)
		b, _ = json.MarshalIndent(mv, "    ", "  ") // only strings and integers left
	}
	return b
}

func metricView(m *pb.Payload_Metric) MetricView {
	mv := MetricView{
		Name: m.GetName(),
		Type: dataTypeName(pb.DataType(m.GetDatatype())),
	}
	if m.Alias != nil {
		mv.Alias = m.GetAlias()
		if mv.Name == "" {
			mv.Name = fmt.Sprintf("alias/%d", mv.Alias)
		}
	}
	if m.Timestamp != nil {
		mv.Timestamp = millisTime(*m.Timestamp)
	}

	val, err := payload.GetValue(m)
	if err == nil {
		mv.Value = val
	}

	q, ok := payload.GetQuality(m)
	if ok {
		mv.Quality = qualityName(q)
	}
	return mv
}

func qualityName(q int32) string {
	switch q {
	case payload.QualityGood:
		return "GOOD"
	case payload.QualityBad:
		return "BAD"
	case payload.QualityStale:
		return "STALE"
	default:
		return fmt.Sprintf("%d", q)
	}
}

// millisTime formats a timestamp of the message, milliseconds since the epoch,
// as RFC 3339 in UTC. One past math.MaxInt64 has no time.Time and is printed
// as its number, rather than wrapping to a time before the epoch.
func millisTime(ms uint64) string {
	if ms > math.MaxInt64 {
		return strconv.FormatUint(ms, 10)
	}
	return time.UnixMilli(int64(ms)).UTC().Format(time.RFC3339)
}

func dataTypeName(dt pb.DataType) string {
	s := dt.String()
	return strings.TrimPrefix(s, "DataType_")
}
