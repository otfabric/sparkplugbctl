// SPDX-License-Identifier: MIT

package main

import (
	"os"

	"github.com/otfabric/sparkplugbctl/cmd"
)

// Injected at link time (Makefile and the release workflow):
// -X main.version=... -X main.commit=... -X main.date=...
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cmd.Version, cmd.Commit, cmd.BuildDate = version, commit, date
	if err := cmd.Execute(); err != nil {
		os.Exit(cmd.ExitCode(err))
	}
}
