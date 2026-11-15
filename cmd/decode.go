// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/otfabric/go-sparkplugb/payload"
	"github.com/otfabric/sparkplugbctl/internal/output"
	"github.com/spf13/cobra"
)

var decodeFile string

var decodeCmd = &cobra.Command{
	ValidArgsFunction: cobra.NoFileCompletions,
	Use:               "decode",
	Short:             "Decode a Sparkplug B Protobuf payload",
	Long: `Decode reads a binary Sparkplug B Protobuf payload from stdin or
a file and pretty-prints or JSON-encodes it.`,
	RunE: runDecode,
}

func init() {
	decodeCmd.Flags().StringVar(&decodeFile, "file", "", "read payload from file (default: stdin)")
	completeFlags(decodeCmd, anyFile, "file")
	addMaxPayloadFlag(decodeCmd)
	rootCmd.AddCommand(decodeCmd)
}

func runDecode(cmd *cobra.Command, args []string) error {
	limit, err := payloadLimit()
	if err != nil {
		return err
	}
	r := cmd.InOrStdin()
	if decodeFile != "" {
		f, err := os.Open(decodeFile)
		if err != nil {
			return fmt.Errorf("open %s: %w", decodeFile, err)
		}
		defer func() { _ = f.Close() }()
		r = f
	}

	// One byte more than the limit tells a payload at the limit from a
	// larger input, without reading the rest of it.
	data, err := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	if len(data) > limit {
		return withExit(ExitInvalidInput, fmt.Errorf("read input: larger than %d bytes (--max-payload)", limit))
	}

	p, err := payload.Decode(data)
	if err != nil {
		return withExit(ExitInvalidInput, fmt.Errorf("decode protobuf: %w", err))
	}

	printer := &output.Printer{W: cmd.OutOrStdout(), JSON: jsonOut}
	return outputError(printer.PrintPayload(p))
}
