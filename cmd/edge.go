// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"os/signal"
	"syscall"
	"time"

	"github.com/otfabric/go-sparkplugb/edge"
	"github.com/otfabric/go-sparkplugb/payload"
	pb "github.com/otfabric/go-sparkplugb/proto"
	"github.com/otfabric/go-sparkplugb/topic"
	"github.com/otfabric/sparkplugbctl/internal/output"
	"github.com/spf13/cobra"
)

var (
	edgeGroup       string
	edgeNode        string
	edgeDevices     []string
	edgeMetrics     []string
	edgeInterval    time.Duration
	edgePrimaryHost string
)

var edgeCmd = &cobra.Command{
	ValidArgsFunction: cobra.NoFileCompletions,
	Use:               "edge",
	Short:             "Simulate a Sparkplug B edge node",
	Long: `Edge simulates a full Sparkplug B edge node lifecycle:
Connect → NBIRTH → DBIRTH(s) → DATA loop → Ctrl+C → DDEATH(s) → NDEATH → Disconnect

Metrics are specified as name:type=value and updated with small random
deltas on each publish interval.`,
	RunE: runEdge,
}

func init() {
	edgeCmd.Flags().StringVar(&edgeGroup, "group", "", "Sparkplug group ID (required; "+idRule+")")
	edgeCmd.Flags().StringVar(&edgeNode, "node", "", "Sparkplug edge node ID (required; "+idRule+")")
	edgeCmd.Flags().StringSliceVar(&edgeDevices, "device", nil, "device IDs (repeatable; "+idRule+")")
	edgeCmd.Flags().StringSliceVar(&edgeMetrics, "metric", nil, "node metrics as name:type=value (repeatable)")
	edgeCmd.Flags().DurationVar(&edgeInterval, "interval", 5*time.Second, "data publish interval")
	edgeCmd.Flags().StringVar(&edgePrimaryHost, "primary-host", "", "primary host application ID")
	completeFlags(edgeCmd, cobra.NoFileCompletions, "group", "node", "device", "interval", "primary-host")
	completeFlags(edgeCmd, completeMetric, "metric")
	_ = edgeCmd.MarkFlagRequired("group")
	_ = edgeCmd.MarkFlagRequired("node")
	rootCmd.AddCommand(edgeCmd)
}

func runEdge(cmd *cobra.Command, args []string) error {
	stderr := cmd.ErrOrStderr()
	// The command line is checked before connecting: the connection arms
	// the node's NDEATH will, which the broker publishes if the command ends
	// without a Disconnect.
	if err := checkTopicID("--group", edgeGroup); err != nil {
		return err
	}
	if err := checkTopicID("--node", edgeNode); err != nil {
		return err
	}
	for _, devID := range edgeDevices {
		if err := checkTopicID("--device", devID); err != nil {
			return err
		}
	}
	if edgeInterval <= 0 {
		return withExit(ExitInvalidInput, fmt.Errorf("--interval %s: must be more than 0", edgeInterval))
	}
	birthMetrics, err := buildEdgeMetrics(edgeMetrics)
	if err != nil {
		return withExit(ExitInvalidInput, err)
	}
	opts, err := connOpts(stderr)
	if err != nil {
		return err
	}
	transport, err := newTransport(opts)
	if err != nil {
		return withExit(ExitInvalidInput, err)
	}

	cid := clientID
	if cid == "" {
		cid = fmt.Sprintf("sparkplugbctl-edge-%s-%s", edgeGroup, edgeNode)
	}

	// The node is Online once connected, or, with a primary host, once that
	// host's STATE is online; the births follow each time. OnState runs with
	// the node's lock held, so it only signals.
	online := make(chan struct{}, 1)
	cfg := edge.Config{
		GroupID:       edgeGroup,
		EdgeNodeID:    edgeNode,
		BrokerURL:     brokerURL,
		ClientID:      cid,
		Username:      opts.Username,
		Password:      opts.Password,
		PrimaryHostID: edgePrimaryHost,
		Logger:        &cliLogger{w: stderr},
		OnState: func(_, s edge.State) {
			if s == edge.Online {
				select {
				case online <- struct{}{}:
				default:
				}
			}
		},
		OnNodeCmd: func(deviceID string, metrics []*pb.Payload_Metric) {
			_, _ = fmt.Fprintf(stderr, "NCMD received: %d metric(s)\n", len(metrics))
		},
		OnDeviceCmd: func(deviceID string, metrics []*pb.Payload_Metric) {
			_, _ = fmt.Fprintf(stderr, "DCMD for %s: %d metric(s)\n", output.Escape(deviceID), len(metrics))
		},
	}

	node, err := edge.New(transport, cfg)
	if err != nil {
		return withExit(ExitInvalidInput, err)
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := node.Connect(ctx); err != nil {
		return withExit(ExitTransport, fmt.Errorf("connect: %w", err))
	}
	_, _ = fmt.Fprintf(stderr, "Connected to %s as %s/%s\n", redactURL(brokerURL), edgeGroup, edgeNode)
	if edgePrimaryHost != "" && node.State() != edge.Online {
		_, _ = fmt.Fprintf(stderr, "Waiting for primary host %s to be online\n", edgePrimaryHost)
	}

	births := func() error {
		if err := node.PublishBirth(ctx, birthMetrics); err != nil {
			return withExit(ExitTransport, fmt.Errorf("publish NBIRTH: %w", err))
		}
		_, _ = fmt.Fprintf(stderr, "Published NBIRTH (%d metrics)\n", len(birthMetrics))
		for _, devID := range edgeDevices {
			devMetric, _ := payload.NewMetric("status", pb.DataType_Boolean, true)
			if err := node.PublishDeviceBirth(ctx, devID, []*pb.Payload_Metric{devMetric}); err != nil {
				return withExit(ExitTransport, fmt.Errorf("publish DBIRTH %s: %w", devID, err))
			}
			_, _ = fmt.Fprintf(stderr, "Published DBIRTH for %s\n", devID)
		}
		return nil
	}

	ticks, stopTicker := newTicker(edgeInterval)
	defer stopTicker()

	values := make(map[string]float64)
	for _, m := range birthMetrics {
		if v, err := payload.GetValue(m); err == nil {
			switch vt := v.(type) {
			case float64:
				values[m.GetName()] = vt
			case float32:
				values[m.GetName()] = float64(vt)
			case int32:
				values[m.GetName()] = float64(vt)
			}
		}
	}

	_, _ = fmt.Fprintf(stderr, "Publishing data every %s (Ctrl+C to stop)\n", edgeInterval)
	for {
		select {
		case <-ctx.Done():
			_, _ = fmt.Fprintln(stderr, "\nShutting down edge node...")
			shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
			for _, devID := range edgeDevices {
				if err := node.PublishDeviceDeath(shutCtx, devID); err != nil {
					_, _ = fmt.Fprintf(stderr, "publish DDEATH %s: %v\n", devID, err)
					continue
				}
				_, _ = fmt.Fprintf(stderr, "Published DDEATH for %s\n", devID)
			}
			_ = node.Disconnect(shutCtx)
			shutCancel()
			_, _ = fmt.Fprintln(stderr, "Disconnected")
			return nil

		case <-online:
			if err := births(); err != nil {
				_, _ = fmt.Fprintf(stderr, "%v\n", err)
			}

		case <-ticks:
			if node.State() != edge.Online {
				continue
			}
			dataMetrics := make([]*pb.Payload_Metric, 0, len(values))
			for name, val := range values {
				delta := (rand.Float64() - 0.5) * 2.0
				val += delta
				values[name] = val
				m, _ := payload.NewMetric(name, pb.DataType_Double, val)
				dataMetrics = append(dataMetrics, m)
			}
			if err := node.PublishNodeData(ctx, dataMetrics); err != nil {
				_, _ = fmt.Fprintf(stderr, "publish NDATA: %v\n", err)
			} else {
				_, _ = fmt.Fprintf(stderr, "Published NDATA (%d metrics)\n", len(dataMetrics))
			}
		}
	}
}

// idRule is the rule for a group, edge node or device ID: the topic segment
// rule of Sparkplug B 3.0.0 (tck-id-topics-device-id).
const idRule = "non-empty valid UTF-8 without '/', '+' or '#'"

// checkTopicID returns an invalid-input error naming flag and id when id
// cannot be a segment of a Sparkplug topic. The check is go-sparkplugb's
// own, the one its edge node applies in New and PublishDeviceBirth: the
// topic package has no exported check of a single ID, so the ID is parsed as
// the last segment of a STATE topic, as go-sparkplugb's edge package does.
func checkTopicID(flag, id string) error {
	if _, err := topic.Parse(topic.ForState(id)); err != nil {
		return withExit(ExitInvalidInput, fmt.Errorf("%s %q: a Sparkplug ID is %s", flag, id, idRule))
	}
	return nil
}

func buildEdgeMetrics(specs []string) ([]*pb.Payload_Metric, error) {
	if len(specs) == 0 {
		m1, _ := payload.NewMetric("temperature", pb.DataType_Double, float64(22.5))
		m2, _ := payload.NewMetric("humidity", pb.DataType_Double, float64(55.0))
		m3, _ := payload.NewMetric("pressure", pb.DataType_Double, float64(1013.25))
		return []*pb.Payload_Metric{m1, m2, m3}, nil
	}
	return parseMetrics(specs)
}

type cliLogger struct {
	w io.Writer
}

func (l *cliLogger) Debugf(format string, args ...any) {
	_, _ = fmt.Fprintf(l.w, "[DEBUG] "+format+"\n", args...)
}
func (l *cliLogger) Infof(format string, args ...any) {
	_, _ = fmt.Fprintf(l.w, "[INFO]  "+format+"\n", args...)
}
func (l *cliLogger) Warnf(format string, args ...any) {
	_, _ = fmt.Fprintf(l.w, "[WARN]  "+format+"\n", args...)
}
func (l *cliLogger) Errorf(format string, args ...any) {
	_, _ = fmt.Fprintf(l.w, "[ERROR] "+format+"\n", args...)
}
