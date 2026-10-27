// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"

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
		_, _ = fmt.Fprintf(w, "  library: go-sparkplugb %s\n", libraryVersion())
		_, _ = fmt.Fprintf(w, "  go:      %s\n", runtime.Version())
		_, _ = fmt.Fprintf(w, "  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

// libraryVersion is the go-sparkplugb module version this binary was built
// with, from its build information.
func libraryVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, dep := range info.Deps {
		if dep.Path == "github.com/otfabric/go-sparkplugb" {
			if dep.Replace != nil {
				return dep.Replace.Version + " (replaced by " + dep.Replace.Path + ")"
			}
			return dep.Version
		}
	}
	return "unknown"
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
