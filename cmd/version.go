// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		w := cmd.OutOrStdout()
		_, _ = fmt.Fprintf(w, "sparkplugbctl %s\n", Version)
		_, _ = fmt.Fprintf(w, "  commit:  %s\n", Commit)
		_, _ = fmt.Fprintf(w, "  built:   %s\n", BuildDate)
		_, _ = fmt.Fprintf(w, "  go:      %s\n", runtime.Version())
		_, _ = fmt.Fprintf(w, "  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
