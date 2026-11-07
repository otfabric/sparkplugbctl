// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	"github.com/otfabric/sparkplugbctl/internal/buildinfo"
	"github.com/spf13/cobra"
)

// Build metadata, set by main from the values injected at link time. Empty
// means not injected: the version command then falls back to the Go build
// information, then to "dev" or "unknown".
var (
	Version   string
	Tag       string
	Commit    string
	BuildDate string
)

var versionCmd = &cobra.Command{
	Use:               "version",
	Short:             "Print version information",
	ValidArgsFunction: cobra.NoFileCompletions,
	Run: func(cmd *cobra.Command, args []string) {
		info := buildinfo.Resolve(cmd.Root().Name(), Version, Tag, Commit, BuildDate)
		line := info.String()
		if jsonOut {
			line = info.JSON()
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
