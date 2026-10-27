// SPDX-License-Identifier: MIT

package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/otfabric/go-sparkplugb/payload"
	pb "github.com/otfabric/go-sparkplugb/proto"
	"github.com/otfabric/go-sparkplugb/topic"
)

type Printer struct {
	W    io.Writer
	JSON bool
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

type MetricView struct {
	Name      string `json:"name"`
	Alias     uint64 `json:"alias,omitempty"`
	Type      string `json:"type"`
	Value     any    `json:"value"`
	Timestamp string `json:"timestamp,omitempty"`
	Quality   string `json:"quality,omitempty"`
}

func (p *Printer) PrintMessage(t topic.Topic, pl *pb.Payload) {
	mv := MessageView{
		Topic:    t.String(),
		Type:     string(t.Type),
		GroupID:  t.GroupID,
		NodeID:   t.EdgeNodeID,
		DeviceID: t.DeviceID,
		Seq:      pl.GetSeq(),
	}
	if pl.Timestamp != nil {
		mv.Time = time.UnixMilli(int64(*pl.Timestamp)).UTC().Format(time.RFC3339)
	}
	for _, m := range pl.Metrics {
		mv.Metrics = append(mv.Metrics, metricView(m))
	}

	if p.JSON {
		enc := json.NewEncoder(p.W)
		enc.SetIndent("", "  ")
		_ = enc.Encode(mv)
		return
	}

	fmt.Fprintf(p.W, "[%s] %s\n", t.Type, t.String())
	fmt.Fprintf(p.W, "  seq=%d", mv.Seq)
	if mv.Time != "" {
		fmt.Fprintf(p.W, "  ts=%s", mv.Time)
	}
	fmt.Fprintln(p.W)

	tw := tabwriter.NewWriter(p.W, 2, 4, 2, ' ', 0)
	for _, m := range mv.Metrics {
		line := fmt.Sprintf("  %s\t%s\t%v", m.Name, m.Type, m.Value)
		if m.Quality != "" {
			line += fmt.Sprintf("\tquality=%s", m.Quality)
		}
		fmt.Fprintln(tw, line)
	}
	tw.Flush()
	fmt.Fprintln(p.W)
}

func (p *Printer) PrintPayload(pl *pb.Payload) {
	if p.JSON {
		mv := struct {
			Seq     uint64       `json:"seq"`
			Time    string       `json:"timestamp,omitempty"`
			Metrics []MetricView `json:"metrics"`
		}{Seq: pl.GetSeq()}
		if pl.Timestamp != nil {
			mv.Time = time.UnixMilli(int64(*pl.Timestamp)).UTC().Format(time.RFC3339)
		}
		for _, m := range pl.Metrics {
			mv.Metrics = append(mv.Metrics, metricView(m))
		}
		enc := json.NewEncoder(p.W)
		enc.SetIndent("", "  ")
		_ = enc.Encode(mv)
		return
	}

	fmt.Fprintf(p.W, "seq=%d", pl.GetSeq())
	if pl.Timestamp != nil {
		fmt.Fprintf(p.W, "  ts=%s", time.UnixMilli(int64(*pl.Timestamp)).UTC().Format(time.RFC3339))
	}
	fmt.Fprintln(p.W)

	tw := tabwriter.NewWriter(p.W, 2, 4, 2, ' ', 0)
	for _, m := range pl.Metrics {
		mv := metricView(m)
		line := fmt.Sprintf("  %s\t%s\t%v", mv.Name, mv.Type, mv.Value)
		if mv.Quality != "" {
			line += fmt.Sprintf("\tquality=%s", mv.Quality)
		}
		fmt.Fprintln(tw, line)
	}
	tw.Flush()
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
		mv.Timestamp = time.UnixMilli(int64(*m.Timestamp)).UTC().Format(time.RFC3339)
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

func dataTypeName(dt pb.DataType) string {
	s := dt.String()
	return strings.TrimPrefix(s, "DataType_")
}
