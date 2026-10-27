// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/otfabric/go-sparkplugb/app"
	pb "github.com/otfabric/go-sparkplugb/proto"
	"github.com/otfabric/sparkplugbctl/internal/broker"
	"github.com/spf13/cobra"
)

var appGroup string

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Run a Sparkplug B application (subscriber, no STATE)",
	Long: `App runs a Sparkplug B application that subscribes to spBv1.0/#,
tracks edge nodes and devices, and displays events as they arrive.

Unlike the host command, an application does NOT publish STATE birth/death
certificates and is not a Primary Host. It passively observes the namespace,
resolves aliases, and can send NCMD/DCMD commands.`,
	RunE: runApp,
}

func init() {
	appCmd.Flags().StringVar(&appGroup, "group", "", "filter events by group ID (optional)")
	rootCmd.AddCommand(appCmd)
}

func runApp(cmd *cobra.Command, args []string) error {
	transport, err := broker.NewTransport(brokerOpts())
	if err != nil {
		return err
	}

	cid := clientID
	if cid == "" {
		cid = fmt.Sprintf("sparkplugbctl-app-%d", time.Now().UnixNano()%100000)
	}

	cfg := app.Config{
		BrokerURL: brokerURL,
		ClientID:  cid,
		Logger:    &cliLogger{},
		OnNodeBirth: func(groupID, nodeID string, metrics []*pb.Payload_Metric) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				printJSON("NBIRTH", groupID, nodeID, "", metrics)
			} else {
				ts := time.Now().Format(time.RFC3339)
				fmt.Fprintf(os.Stdout, "%s  [NBIRTH] %s/%s (%d metrics)\n", ts, groupID, nodeID, len(metrics))
			}
		},
		OnDeviceBirth: func(groupID, nodeID, deviceID string, metrics []*pb.Payload_Metric) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				printJSON("DBIRTH", groupID, nodeID, deviceID, metrics)
			} else {
				ts := time.Now().Format(time.RFC3339)
				fmt.Fprintf(os.Stdout, "%s  [DBIRTH] %s/%s/%s (%d metrics)\n", ts, groupID, nodeID, deviceID, len(metrics))
			}
		},
		OnNodeData: func(groupID, nodeID string, metrics []*pb.Payload_Metric) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				printJSON("NDATA", groupID, nodeID, "", metrics)
			} else {
				ts := time.Now().Format(time.RFC3339)
				fmt.Fprintf(os.Stdout, "%s  [NDATA]  %s/%s (%d metrics)\n", ts, groupID, nodeID, len(metrics))
			}
		},
		OnDeviceData: func(groupID, nodeID, deviceID string, metrics []*pb.Payload_Metric) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				printJSON("DDATA", groupID, nodeID, deviceID, metrics)
			} else {
				ts := time.Now().Format(time.RFC3339)
				fmt.Fprintf(os.Stdout, "%s  [DDATA]  %s/%s/%s (%d metrics)\n", ts, groupID, nodeID, deviceID, len(metrics))
			}
		},
		OnNodeDeath: func(groupID, nodeID string) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				printJSON("NDEATH", groupID, nodeID, "", nil)
			} else {
				ts := time.Now().Format(time.RFC3339)
				fmt.Fprintf(os.Stdout, "%s  [NDEATH] %s/%s\n", ts, groupID, nodeID)
			}
		},
		OnDeviceDeath: func(groupID, nodeID, deviceID string) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				printJSON("DDEATH", groupID, nodeID, deviceID, nil)
			} else {
				ts := time.Now().Format(time.RFC3339)
				fmt.Fprintf(os.Stdout, "%s  [DDEATH] %s/%s/%s\n", ts, groupID, nodeID, deviceID)
			}
		},
	}

	a, err := app.New(transport, cfg)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := a.Connect(ctx); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Application mode (no STATE) — broker: %s\n", brokerURL)
	if appGroup != "" {
		fmt.Fprintf(os.Stderr, "Filtering: group=%s\n", appGroup)
	}
	fmt.Fprintf(os.Stderr, "Listening for edge activity (Ctrl+C to stop)\n")

	<-ctx.Done()
	fmt.Fprintln(os.Stderr, "\nShutting down application...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = a.Close(shutCtx)
	fmt.Fprintln(os.Stderr, "Disconnected")
	return nil
}

type appEvent struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	GroupID   string `json:"group_id"`
	NodeID    string `json:"node_id"`
	DeviceID  string `json:"device_id,omitempty"`
	Metrics   int    `json:"metrics,omitempty"`
}

func printJSON(msgType, groupID, nodeID, deviceID string, metrics []*pb.Payload_Metric) {
	ev := appEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      msgType,
		GroupID:   groupID,
		NodeID:    nodeID,
		DeviceID:  deviceID,
		Metrics:   len(metrics),
	}
	b, _ := json.Marshal(ev)
	fmt.Fprintln(os.Stdout, string(b))
}
