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
	"github.com/otfabric/sparkplugbctl/internal/output"
	"github.com/spf13/cobra"
)

var appGroup string

var appCmd = &cobra.Command{
	ValidArgsFunction: cobra.NoFileCompletions,
	Use:               "app",
	Short:             "Run a Sparkplug B application (subscriber, no STATE)",
	Long: `App runs a Sparkplug B application that subscribes to spBv1.0/#,
tracks edge nodes and devices, and displays events as they arrive.

Unlike the host command, an application does NOT publish STATE birth/death
certificates and is not a Primary Host. It passively observes the namespace
and resolves aliases.`,
	RunE: runApp,
}

func init() {
	appCmd.Flags().StringVar(&appGroup, "group", "", "filter events by group ID (optional)")
	completeFlags(appCmd, cobra.NoFileCompletions, "group")
	rootCmd.AddCommand(appCmd)
}

func runApp(cmd *cobra.Command, args []string) error {
	stdout, stderr := cmd.OutOrStdout(), cmd.ErrOrStderr()
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
		cid = randomClientID("app")
	}

	out := newEventWriter(stdout)
	cfg := app.Config{
		BrokerURL: brokerURL,
		ClientID:  cid,
		Username:  opts.Username,
		Password:  opts.Password,
		Logger:    &cliLogger{w: stderr},
		OnNodeBirth: func(groupID, nodeID string, metrics []*pb.Payload_Metric) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				out.report(printJSON(stdout, "NBIRTH", groupID, nodeID, "", metrics))
			} else {
				ts := time.Now().Format(time.RFC3339)
				out.printf("%s  [NBIRTH] %s/%s (%d metrics)\n", ts, output.Escape(groupID), output.Escape(nodeID), len(metrics))
			}
		},
		OnDeviceBirth: func(groupID, nodeID, deviceID string, metrics []*pb.Payload_Metric) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				out.report(printJSON(stdout, "DBIRTH", groupID, nodeID, deviceID, metrics))
			} else {
				ts := time.Now().Format(time.RFC3339)
				out.printf("%s  [DBIRTH] %s/%s/%s (%d metrics)\n", ts, output.Escape(groupID), output.Escape(nodeID), output.Escape(deviceID), len(metrics))
			}
		},
		OnNodeData: func(groupID, nodeID string, metrics []*pb.Payload_Metric) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				out.report(printJSON(stdout, "NDATA", groupID, nodeID, "", metrics))
			} else {
				ts := time.Now().Format(time.RFC3339)
				out.printf("%s  [NDATA]  %s/%s (%d metrics)\n", ts, output.Escape(groupID), output.Escape(nodeID), len(metrics))
			}
		},
		OnDeviceData: func(groupID, nodeID, deviceID string, metrics []*pb.Payload_Metric) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				out.report(printJSON(stdout, "DDATA", groupID, nodeID, deviceID, metrics))
			} else {
				ts := time.Now().Format(time.RFC3339)
				out.printf("%s  [DDATA]  %s/%s/%s (%d metrics)\n", ts, output.Escape(groupID), output.Escape(nodeID), output.Escape(deviceID), len(metrics))
			}
		},
		OnNodeDeath: func(groupID, nodeID string) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				out.report(printJSON(stdout, "NDEATH", groupID, nodeID, "", nil))
			} else {
				ts := time.Now().Format(time.RFC3339)
				out.printf("%s  [NDEATH] %s/%s\n", ts, output.Escape(groupID), output.Escape(nodeID))
			}
		},
		OnDeviceDeath: func(groupID, nodeID, deviceID string) {
			if appGroup != "" && groupID != appGroup {
				return
			}
			if jsonOut {
				out.report(printJSON(stdout, "DDEATH", groupID, nodeID, deviceID, nil))
			} else {
				ts := time.Now().Format(time.RFC3339)
				out.printf("%s  [DDEATH] %s/%s/%s\n", ts, output.Escape(groupID), output.Escape(nodeID), output.Escape(deviceID))
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

	_, _ = fmt.Fprintf(stderr, "Application mode (no STATE) — broker: %s\n", redactURL(brokerURL))
	if appGroup != "" {
		_, _ = fmt.Fprintf(stderr, "Filtering: group=%s\n", appGroup)
	}
	_, _ = fmt.Fprintf(stderr, "Listening for edge activity (Ctrl+C to stop)\n")

	var outErr error
	select {
	case <-ctx.Done():
	case outErr = <-out.failed:
	}
	_, _ = fmt.Fprintln(stderr, "\nShutting down application...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = a.Close(shutCtx)
	_, _ = fmt.Fprintln(stderr, "Disconnected")
	return outputError(outErr)
}

type appEvent struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	GroupID   string `json:"group_id"`
	NodeID    string `json:"node_id"`
	DeviceID  string `json:"device_id,omitempty"`
	Metrics   int    `json:"metrics,omitempty"`
}

func printJSON(w io.Writer, msgType, groupID, nodeID, deviceID string, metrics []*pb.Payload_Metric) (int, error) {
	ev := appEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      msgType,
		GroupID:   groupID,
		NodeID:    nodeID,
		DeviceID:  deviceID,
		Metrics:   len(metrics),
	}
	b, _ := json.Marshal(ev) // strings and integers only
	return fmt.Fprintln(w, string(b))
}
