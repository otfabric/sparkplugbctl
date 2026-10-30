// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/signal"
	"syscall"
	"time"

	"github.com/otfabric/go-sparkplugb/app"
	pb "github.com/otfabric/go-sparkplugb/proto"
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
	stdout, stderr := cmd.OutOrStdout(), cmd.ErrOrStderr()
	transport, err := newTransport(brokerOpts())
	if err != nil {
		return withExit(ExitInvalidInput, err)
	}

	cid := clientID
	if cid == "" {
		cid = fmt.Sprintf("sparkplugbctl-app-%d", time.Now().UnixNano()%100000)
	}

	cfg := app.Config{
		BrokerURL: brokerURL,
		ClientID:  cid,
		Logger:    &cliLogger{w: stderr},
		OnNodeBirth: func(groupID, nodeID string, metrics []*pb.Payload_Metric) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				printJSON(stdout, "NBIRTH", groupID, nodeID, "", metrics)
			} else {
				ts := time.Now().Format(time.RFC3339)
				_, _ = fmt.Fprintf(stdout, "%s  [NBIRTH] %s/%s (%d metrics)\n", ts, groupID, nodeID, len(metrics))
			}
		},
		OnDeviceBirth: func(groupID, nodeID, deviceID string, metrics []*pb.Payload_Metric) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				printJSON(stdout, "DBIRTH", groupID, nodeID, deviceID, metrics)
			} else {
				ts := time.Now().Format(time.RFC3339)
				_, _ = fmt.Fprintf(stdout, "%s  [DBIRTH] %s/%s/%s (%d metrics)\n", ts, groupID, nodeID, deviceID, len(metrics))
			}
		},
		OnNodeData: func(groupID, nodeID string, metrics []*pb.Payload_Metric) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				printJSON(stdout, "NDATA", groupID, nodeID, "", metrics)
			} else {
				ts := time.Now().Format(time.RFC3339)
				_, _ = fmt.Fprintf(stdout, "%s  [NDATA]  %s/%s (%d metrics)\n", ts, groupID, nodeID, len(metrics))
			}
		},
		OnDeviceData: func(groupID, nodeID, deviceID string, metrics []*pb.Payload_Metric) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				printJSON(stdout, "DDATA", groupID, nodeID, deviceID, metrics)
			} else {
				ts := time.Now().Format(time.RFC3339)
				_, _ = fmt.Fprintf(stdout, "%s  [DDATA]  %s/%s/%s (%d metrics)\n", ts, groupID, nodeID, deviceID, len(metrics))
			}
		},
		OnNodeDeath: func(groupID, nodeID string) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				printJSON(stdout, "NDEATH", groupID, nodeID, "", nil)
			} else {
				ts := time.Now().Format(time.RFC3339)
				_, _ = fmt.Fprintf(stdout, "%s  [NDEATH] %s/%s\n", ts, groupID, nodeID)
			}
		},
		OnDeviceDeath: func(groupID, nodeID, deviceID string) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				printJSON(stdout, "DDEATH", groupID, nodeID, deviceID, nil)
			} else {
				ts := time.Now().Format(time.RFC3339)
				_, _ = fmt.Fprintf(stdout, "%s  [DDEATH] %s/%s/%s\n", ts, groupID, nodeID, deviceID)
			}
		},
	}

	a, err := app.New(transport, cfg)
	if err != nil {
		return withExit(ExitInvalidInput, err)
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := a.Connect(ctx); err != nil {
		return withExit(ExitTransport, fmt.Errorf("connect: %w", err))
	}

	_, _ = fmt.Fprintf(stderr, "Application mode (no STATE) — broker: %s\n", brokerURL)
	if appGroup != "" {
		_, _ = fmt.Fprintf(stderr, "Filtering: group=%s\n", appGroup)
	}
	_, _ = fmt.Fprintf(stderr, "Listening for edge activity (Ctrl+C to stop)\n")

	<-ctx.Done()
	_, _ = fmt.Fprintln(stderr, "\nShutting down application...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = a.Close(shutCtx)
	_, _ = fmt.Fprintln(stderr, "Disconnected")
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

func printJSON(w io.Writer, msgType, groupID, nodeID, deviceID string, metrics []*pb.Payload_Metric) {
	ev := appEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      msgType,
		GroupID:   groupID,
		NodeID:    nodeID,
		DeviceID:  deviceID,
		Metrics:   len(metrics),
	}
	b, _ := json.Marshal(ev)
	_, _ = fmt.Fprintln(w, string(b))
}
