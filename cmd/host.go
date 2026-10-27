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
	"github.com/otfabric/sparkplugbctl/internal/broker"
	"github.com/spf13/cobra"
)

var hostID string

var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "Simulate a Sparkplug B host application",
	Long: `Host simulates a Sparkplug B host application:
Connect → STATE ONLINE → listen for edge activity → Ctrl+C → STATE OFFLINE → Disconnect`,
	RunE: runHost,
}

func init() {
	hostCmd.Flags().StringVar(&hostID, "host-id", "", "host application ID (required)")
	_ = hostCmd.MarkFlagRequired("host-id")
	rootCmd.AddCommand(hostCmd)
}

func runHost(cmd *cobra.Command, args []string) error {
	stdout, stderr := cmd.OutOrStdout(), cmd.ErrOrStderr()
	transport, err := broker.NewTransport(brokerOpts())
	if err != nil {
		return err
	}

	cid := clientID
	if cid == "" {
		cid = fmt.Sprintf("sparkplugbctl-host-%s", hostID)
	}

	cfg := host.Config{
		HostID:    hostID,
		BrokerURL: brokerURL,
		ClientID:  cid,
		Logger:    &cliLogger{w: stderr},
		OnNodeBirth: func(groupID, nodeID string, metrics []*pb.Payload_Metric) {
			_, _ = fmt.Fprintf(stdout, "[NBIRTH] %s/%s (%d metrics)\n", groupID, nodeID, len(metrics))
		},
		OnDeviceBirth: func(groupID, nodeID, deviceID string, metrics []*pb.Payload_Metric) {
			_, _ = fmt.Fprintf(stdout, "[DBIRTH] %s/%s/%s (%d metrics)\n", groupID, nodeID, deviceID, len(metrics))
		},
		OnNodeData: func(groupID, nodeID string, metrics []*pb.Payload_Metric) {
			_, _ = fmt.Fprintf(stdout, "[NDATA]  %s/%s (%d metrics)\n", groupID, nodeID, len(metrics))
		},
		OnDeviceData: func(groupID, nodeID, deviceID string, metrics []*pb.Payload_Metric) {
			_, _ = fmt.Fprintf(stdout, "[DDATA]  %s/%s/%s (%d metrics)\n", groupID, nodeID, deviceID, len(metrics))
		},
		OnNodeDeath: func(groupID, nodeID string) {
			_, _ = fmt.Fprintf(stdout, "[NDEATH] %s/%s\n", groupID, nodeID)
		},
		OnDeviceDeath: func(groupID, nodeID, deviceID string) {
			_, _ = fmt.Fprintf(stdout, "[DDEATH] %s/%s/%s\n", groupID, nodeID, deviceID)
		},
	}

	app, err := host.New(transport, cfg)
	if err != nil {
		return err
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
		return fmt.Errorf("connect: %w", err)
	}

	_, _ = fmt.Fprintf(stderr, "Host %s online (broker: %s)\n", hostID, brokerURL)
	_, _ = fmt.Fprintf(stderr, "Listening for edge activity (Ctrl+C to stop)\n")

	<-ctx.Done()
	_, _ = fmt.Fprintln(stderr, "\nShutting down host...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = app.Disconnect(shutCtx)
	_, _ = fmt.Fprintln(stderr, "Disconnected")
	return nil
}

// Ensure the host command uses the shared transport.Connect path via host.App,
// not a separate transport.Connect call — host.App.Connect handles STATE and Will.
var _ mqtt.Transport = (*mqtt.PahoTransport)(nil)
