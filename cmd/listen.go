// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/otfabric/go-sparkplugb/mqtt"
	"github.com/otfabric/go-sparkplugb/payload"
	"github.com/otfabric/go-sparkplugb/topic"
	"github.com/otfabric/sparkplugbctl/internal/output"
	"github.com/spf13/cobra"
)

var (
	filterGroup  string
	filterNode   string
	filterDevice string
)

var listenCmd = &cobra.Command{
	ValidArgsFunction: cobra.NoFileCompletions,
	Use:               "listen",
	Short:             "Subscribe to Sparkplug B messages and print them",
	Long: `Listen subscribes to spBv1.0/# and decodes incoming Sparkplug B
messages in real time. Use --group, --node, and --device to filter.`,
	RunE: runListen,
}

func init() {
	listenCmd.Flags().StringVar(&filterGroup, "group", "", "filter by group ID")
	listenCmd.Flags().StringVar(&filterNode, "node", "", "filter by edge node ID")
	listenCmd.Flags().StringVar(&filterDevice, "device", "", "filter by device ID")
	completeFlags(listenCmd, cobra.NoFileCompletions, "group", "node", "device")
	addMaxPayloadFlag(listenCmd)
	rootCmd.AddCommand(listenCmd)
}

func runListen(cmd *cobra.Command, args []string) error {
	stdout, stderr := cmd.OutOrStdout(), cmd.ErrOrStderr()
	limit, err := payloadLimit()
	if err != nil {
		return err
	}
	opts, err := connOpts(stderr)
	if err != nil {
		return err
	}
	transport, err := newTransport(opts)
	if err != nil {
		return withExit(ExitInvalidInput, err)
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cid := clientID
	if cid == "" {
		cid = randomClientID("listen")
	}

	lost := make(chan error, 1)
	err = transport.Connect(ctx, mqtt.ConnectOpts{
		BrokerURL:        brokerURL,
		ClientID:         cid,
		CleanSession:     true,
		Username:         opts.Username,
		Password:         opts.Password,
		OnConnectionLost: func(err error) { lost <- err }, // called once
	})
	if err != nil {
		return withExit(ExitTransport, fmt.Errorf("connect: %w", err))
	}
	defer func() { _ = transport.Disconnect(context.Background()) }()

	printer := &output.Printer{W: stdout, JSON: jsonOut}
	outFailed := make(chan error, 1)
	printed := func(err error) {
		if err != nil {
			select {
			case outFailed <- err:
			default:
			}
		}
	}
	decodeError := func(topicStr string, err error) {
		_, _ = fmt.Fprintf(stderr, "decode error on %s: %s\n", output.Escape(topicStr), output.Escape(err.Error()))
	}

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
		if len(data) > limit {
			decodeError(topicStr, fmt.Errorf("payload of %d bytes is above the limit of %d bytes (--max-payload)", len(data), limit))
			return
		}

		if t.Type == topic.STATE {
			err := printer.PrintState(t, data)
			var we *output.WriteError
			if errors.As(err, &we) {
				printed(err)
			} else if err != nil {
				decodeError(topicStr, err)
			}
			return
		}

		p, err := payload.Decode(data)
		if err != nil {
			decodeError(topicStr, err)
			return
		}
		printed(printer.PrintMessage(t, p))
	}

	subTopic := topic.Namespace + "/#"
	if err := transport.Subscribe(ctx, subTopic, 0, handler); err != nil {
		return withExit(ExitTransport, fmt.Errorf("subscribe: %w", err))
	}

	_, _ = fmt.Fprintf(stderr, "Listening on %s (broker: %s)\n", subTopic, redactURL(brokerURL))
	select {
	case err := <-lost:
		return withExit(ExitTransport, fmt.Errorf("listen: %w", err))
	case err := <-outFailed:
		return outputError(err)
	case <-ctx.Done():
	}
	_, _ = fmt.Fprintln(stderr, "\nShutting down...")
	return nil
}
