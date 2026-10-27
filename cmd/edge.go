// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/otfabric/go-sparkplugb/edge"
	"github.com/otfabric/go-sparkplugb/payload"
	pb "github.com/otfabric/go-sparkplugb/proto"
	"github.com/otfabric/sparkplugbctl/internal/broker"
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
	Use:   "edge",
	Short: "Simulate a Sparkplug B edge node",
	Long: `Edge simulates a full Sparkplug B edge node lifecycle:
Connect → NBIRTH → DBIRTH(s) → DATA loop → Ctrl+C → DDEATH(s) → NDEATH → Disconnect

Metrics are specified as name:type=value and updated with small random
deltas on each publish interval.`,
	RunE: runEdge,
}

func init() {
	edgeCmd.Flags().StringVar(&edgeGroup, "group", "", "Sparkplug group ID (required)")
	edgeCmd.Flags().StringVar(&edgeNode, "node", "", "Sparkplug edge node ID (required)")
	edgeCmd.Flags().StringSliceVar(&edgeDevices, "device", nil, "device IDs (repeatable)")
	edgeCmd.Flags().StringSliceVar(&edgeMetrics, "metric", nil, "node metrics as name:type=value (repeatable)")
	edgeCmd.Flags().DurationVar(&edgeInterval, "interval", 5*time.Second, "data publish interval")
	edgeCmd.Flags().StringVar(&edgePrimaryHost, "primary-host", "", "primary host application ID")
	_ = edgeCmd.MarkFlagRequired("group")
	_ = edgeCmd.MarkFlagRequired("node")
	rootCmd.AddCommand(edgeCmd)
}

func runEdge(cmd *cobra.Command, args []string) error {
	transport, err := broker.NewTransport(brokerOpts())
	if err != nil {
		return err
	}

	cid := clientID
	if cid == "" {
		cid = fmt.Sprintf("sparkplugbctl-edge-%s-%s", edgeGroup, edgeNode)
	}

	cfg := edge.Config{
		GroupID:       edgeGroup,
		EdgeNodeID:    edgeNode,
		BrokerURL:     brokerURL,
		ClientID:      cid,
		PrimaryHostID: edgePrimaryHost,
		Logger:        &cliLogger{},
		OnNodeCmd: func(deviceID string, metrics []*pb.Payload_Metric) {
			fmt.Fprintf(os.Stderr, "NCMD received: %d metric(s)\n", len(metrics))
		},
		OnDeviceCmd: func(deviceID string, metrics []*pb.Payload_Metric) {
			fmt.Fprintf(os.Stderr, "DCMD for %s: %d metric(s)\n", deviceID, len(metrics))
		},
	}

	node, err := edge.New(transport, cfg)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := node.Connect(ctx); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Connected to %s as %s/%s\n", brokerURL, edgeGroup, edgeNode)

	birthMetrics, err := buildEdgeMetrics(edgeMetrics)
	if err != nil {
		return err
	}

	if err := node.PublishBirth(ctx, birthMetrics); err != nil {
		return fmt.Errorf("publish NBIRTH: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Published NBIRTH (%d metrics)\n", len(birthMetrics))

	for _, devID := range edgeDevices {
		devMetric, _ := payload.NewMetric("status", pb.DataType_Boolean, true)
		if err := node.PublishDeviceBirth(ctx, devID, []*pb.Payload_Metric{devMetric}); err != nil {
			return fmt.Errorf("publish DBIRTH %s: %w", devID, err)
		}
		fmt.Fprintf(os.Stderr, "Published DBIRTH for %s\n", devID)
	}

	ticker := time.NewTicker(edgeInterval)
	defer ticker.Stop()

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

	fmt.Fprintf(os.Stderr, "Publishing data every %s (Ctrl+C to stop)\n", edgeInterval)
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "\nShutting down edge node...")
			shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutCancel()
			for _, devID := range edgeDevices {
				_ = node.PublishDeviceDeath(shutCtx, devID)
				fmt.Fprintf(os.Stderr, "Published DDEATH for %s\n", devID)
			}
			_ = node.Disconnect(shutCtx)
			fmt.Fprintln(os.Stderr, "Disconnected")
			return nil

		case <-ticker.C:
			dataMetrics := make([]*pb.Payload_Metric, 0, len(values))
			for name, val := range values {
				delta := (rand.Float64() - 0.5) * 2.0
				val += delta
				values[name] = val
				m, _ := payload.NewMetric(name, pb.DataType_Double, val)
				dataMetrics = append(dataMetrics, m)
			}
			if err := node.PublishNodeData(ctx, dataMetrics); err != nil {
				fmt.Fprintf(os.Stderr, "publish NDATA: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "Published NDATA (%d metrics)\n", len(dataMetrics))
			}
		}
	}
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

type cliLogger struct{}

func (l *cliLogger) Debugf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
}
func (l *cliLogger) Infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[INFO]  "+format+"\n", args...)
}
func (l *cliLogger) Warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[WARN]  "+format+"\n", args...)
}
func (l *cliLogger) Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}

