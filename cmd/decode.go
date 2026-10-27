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
	Use:   "decode",
	Short: "Decode a Sparkplug B Protobuf payload",
	Long: `Decode reads a binary Sparkplug B Protobuf payload from stdin or
a file and pretty-prints or JSON-encodes it.`,
	RunE: runDecode,
}

func init() {
	decodeCmd.Flags().StringVar(&decodeFile, "file", "", "read payload from file (default: stdin)")
	rootCmd.AddCommand(decodeCmd)
}

func runDecode(cmd *cobra.Command, args []string) error {
	r := cmd.InOrStdin()
	if decodeFile != "" {
		f, err := os.Open(decodeFile)
		if err != nil {
			return fmt.Errorf("open %s: %w", decodeFile, err)
		}
		defer func() { _ = f.Close() }()
		r = f
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	p, err := payload.Decode(data)
	if err != nil {
		return fmt.Errorf("decode protobuf: %w", err)
	}

	printer := &output.Printer{W: cmd.OutOrStdout(), JSON: jsonOut}
	printer.PrintPayload(p)
	return nil
}
