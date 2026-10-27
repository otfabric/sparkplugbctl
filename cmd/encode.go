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
	Use:   "encode",
	Short: "Encode a Sparkplug B payload to Protobuf",
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
	rootCmd.AddCommand(encodeCmd)
}

func runEncode(cmd *cobra.Command, args []string) error {
	metrics, err := parseMetrics(encodeMetrics)
	if err != nil {
		return err
	}

	p := payload.NewPayload(encodeSeq, metrics...)
	data, err := payload.Encode(p)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if encodeFile != "" {
		if err := os.WriteFile(encodeFile, data, 0644); err != nil {
			return fmt.Errorf("write %s: %w", encodeFile, err)
		}
		_, _ = fmt.Fprintf(os.Stderr, "Wrote %d bytes to %s\n", len(data), encodeFile)
		return nil
	}

	_, err = os.Stdout.Write(data)
	return err
}
