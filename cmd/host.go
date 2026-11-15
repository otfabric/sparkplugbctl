// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/otfabric/go-sparkplugb/host"
	"github.com/otfabric/go-sparkplugb/mqtt"
	pb "github.com/otfabric/go-sparkplugb/proto"
	"github.com/otfabric/sparkplugbctl/internal/output"
	"github.com/spf13/cobra"
)

var hostID string

var hostCmd = &cobra.Command{
	ValidArgsFunction: cobra.NoFileCompletions,
	Use:               "host",
	Short:             "Simulate a Sparkplug B host application",
	Long: `Host simulates a Sparkplug B host application:
Connect → STATE ONLINE → listen for edge activity → Ctrl+C → STATE OFFLINE → Disconnect`,
	RunE: runHost,
}

func init() {
	hostCmd.Flags().StringVar(&hostID, "host-id", "", "host application ID (required)")
	completeFlags(hostCmd, cobra.NoFileCompletions, "host-id")
	_ = hostCmd.MarkFlagRequired("host-id")
	rootCmd.AddCommand(hostCmd)
}

func runHost(cmd *cobra.Command, args []string) error {
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
		cid = fmt.Sprintf("sparkplugbctl-host-%s", hostID)
	}

	out := newEventWriter(stdout)
	cfg := host.Config{
		HostID:    hostID,
		BrokerURL: brokerURL,
		ClientID:  cid,
		Username:  opts.Username,
		Password:  opts.Password,
		Logger:    &cliLogger{w: stderr},
		OnNodeBirth: func(groupID, nodeID string, metrics []*pb.Payload_Metric) {
			out.printf("[NBIRTH] %s/%s (%d metrics)\n", output.Escape(groupID), output.Escape(nodeID), len(metrics))
		},
		OnDeviceBirth: func(groupID, nodeID, deviceID string, metrics []*pb.Payload_Metric) {
			out.printf("[DBIRTH] %s/%s/%s (%d metrics)\n", output.Escape(groupID), output.Escape(nodeID), output.Escape(deviceID), len(metrics))
		},
		OnNodeData: func(groupID, nodeID string, metrics []*pb.Payload_Metric) {
			out.printf("[NDATA]  %s/%s (%d metrics)\n", output.Escape(groupID), output.Escape(nodeID), len(metrics))
		},
		OnDeviceData: func(groupID, nodeID, deviceID string, metrics []*pb.Payload_Metric) {
			out.printf("[DDATA]  %s/%s/%s (%d metrics)\n", output.Escape(groupID), output.Escape(nodeID), output.Escape(deviceID), len(metrics))
		},
		OnNodeDeath: func(groupID, nodeID string) {
			out.printf("[NDEATH] %s/%s\n", output.Escape(groupID), output.Escape(nodeID))
		},
		OnDeviceDeath: func(groupID, nodeID, deviceID string) {
			out.printf("[DDEATH] %s/%s/%s\n", output.Escape(groupID), output.Escape(nodeID), output.Escape(deviceID))
		},
	}

	app, err := host.New(transport, cfg)
	if err != nil {
		return withExit(ExitInvalidInput, err)
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	_ = transport // transport is passed to host.New above

	err = func() error {
		return app.Connect(ctx)
	}()
	if err != nil {
		// host.Connect uses transport internally; we need to connect transport first
		// Actually, host.App.Connect calls transport.Connect itself
		return withExit(ExitTransport, fmt.Errorf("connect: %w", err))
	}

	_, _ = fmt.Fprintf(stderr, "Host %s online (broker: %s)\n", hostID, redactURL(brokerURL))
	_, _ = fmt.Fprintf(stderr, "Listening for edge activity (Ctrl+C to stop)\n")

	var outErr error
	select {
	case <-ctx.Done():
	case outErr = <-out.failed:
	}
	_, _ = fmt.Fprintln(stderr, "\nShutting down host...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = app.Disconnect(shutCtx)
	_, _ = fmt.Fprintln(stderr, "Disconnected")
	return outputError(outErr)
}

// Ensure the host command uses the shared transport.Connect path via host.App,
// not a separate transport.Connect call — host.App.Connect handles STATE and Will.
var _ mqtt.Transport = (*mqtt.PahoTransport)(nil)
