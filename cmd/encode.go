// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"

	"github.com/otfabric/go-sparkplugb/payload"
	"github.com/spf13/cobra"
)

var (
	encodeMetrics []string
	encodeSeq     uint64
	encodeFile    string
)

var encodeCmd = &cobra.Command{
	ValidArgsFunction: cobra.NoFileCompletions,
	Use:               "encode",
	Short:             "Encode a Sparkplug B payload to Protobuf",
	Long: `Encode creates a Sparkplug B Protobuf payload from the given metrics
and writes it to stdout or a file.

Metric format: name:type=value
Examples:
  --metric "temperature:Double=22.5"
  --metric "status:Boolean=true"`,
	RunE: runEncode,
}

func init() {
	encodeCmd.Flags().StringSliceVar(&encodeMetrics, "metric", nil, "metric in name:type=value format (repeatable)")
	encodeCmd.Flags().Uint64Var(&encodeSeq, "seq", 0, "sequence number")
	encodeCmd.Flags().StringVar(&encodeFile, "file", "", "write output to file (default: stdout)")
	completeFlags(encodeCmd, cobra.NoFileCompletions, "seq")
	completeFlags(encodeCmd, completeMetric, "metric")
	completeFlags(encodeCmd, anyFile, "file")
	rootCmd.AddCommand(encodeCmd)
}

func runEncode(cmd *cobra.Command, args []string) error {
	metrics, err := parseMetrics(encodeMetrics)
	if err != nil {
		return withExit(ExitInvalidInput, err)
	}

	p := payload.NewPayload(encodeSeq, metrics...)
	data, err := encodePayload(p)
	if err != nil {
		return withExit(ExitInvalidInput, fmt.Errorf("encode: %w", err))
	}

	if encodeFile != "" {
		if err := os.WriteFile(encodeFile, data, 0o600); err != nil {
			return withExit(ExitInternal, fmt.Errorf("write %s: %w", encodeFile, err))
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Wrote %d bytes to %s\n", len(data), encodeFile)
		return nil
	}

	_, err = cmd.OutOrStdout().Write(data)
	return err
}
