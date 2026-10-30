// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/otfabric/go-sparkplugb/mqtt"
	"github.com/otfabric/go-sparkplugb/payload"
	"github.com/otfabric/go-sparkplugb/topic"
	"github.com/otfabric/sparkplugbctl/internal/broker"
	"github.com/otfabric/sparkplugbctl/internal/output"
	"github.com/spf13/cobra"
)

var (
	filterGroup  string
	filterNode   string
	filterDevice string
)

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Subscribe to Sparkplug B messages and print them",
	Long: `Listen subscribes to spBv1.0/# and decodes incoming Sparkplug B
messages in real time. Use --group, --node, and --device to filter.`,
	RunE: runListen,
}

func init() {
	listenCmd.Flags().StringVar(&filterGroup, "group", "", "filter by group ID")
	listenCmd.Flags().StringVar(&filterNode, "node", "", "filter by edge node ID")
	listenCmd.Flags().StringVar(&filterDevice, "device", "", "filter by device ID")
	rootCmd.AddCommand(listenCmd)
}

func runListen(cmd *cobra.Command, args []string) error {
	stdout, stderr := cmd.OutOrStdout(), cmd.ErrOrStderr()
	transport, err := newTransport(brokerOpts())
	if err != nil {
		return withExit(ExitInvalidInput, err)
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cid := clientID
	if cid == "" {
		cid = "sparkplugbctl-listen"
	}

	err = transport.Connect(ctx, mqtt.ConnectOpts{
		BrokerURL:    brokerURL,
		ClientID:     cid,
		CleanSession: true,
		Username:     username,
		Password:     password,
	})
	if err != nil {
		return withExit(ExitTransport, fmt.Errorf("connect: %w", err))
	}
	defer func() { _ = transport.Disconnect(context.Background()) }()

	printer := &output.Printer{W: stdout, JSON: jsonOut}

	handler := func(topicStr string, data []byte) {
		t, err := topic.Parse(topicStr)
		if err != nil {
			return
		}
		if filterGroup != "" && t.GroupID != filterGroup {
			return
		}
		if filterNode != "" && t.EdgeNodeID != filterNode {
			return
		}
		if filterDevice != "" && t.DeviceID != filterDevice {
			return
		}

		if t.Type == topic.STATE {
			if err := printer.PrintState(t, data); err != nil {
				_, _ = fmt.Fprintf(stderr, "decode error on %s: %v\n", topicStr, err)
			}
			return
		}

		p, err := payload.Decode(data)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "decode error on %s: %v\n", topicStr, err)
			return
		}
		printer.PrintMessage(t, p)
	}

	subTopic := topic.Namespace + "/#"
	if err := transport.Subscribe(ctx, subTopic, 0, handler); err != nil {
		return withExit(ExitTransport, fmt.Errorf("subscribe: %w", err))
	}

	_, _ = fmt.Fprintf(stderr, "Listening on %s (broker: %s)\n", subTopic, brokerURL)
	<-ctx.Done()
	_, _ = fmt.Fprintln(stderr, "\nShutting down...")
	return nil
}

func brokerOpts() broker.Opts {
	return broker.Opts{
		BrokerURL: brokerURL,
		ClientID:  clientID,
		Username:  username,
		Password:  password,
		TLSCert:   tlsCert,
		TLSKey:    tlsKey,
		TLSCA:     tlsCA,
	}
}
