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

// exit ends the process; a test replaces it.
var exit = os.Exit

func main() {
	cmd.Version, cmd.Commit, cmd.BuildDate = version, commit, date
	exit(cmd.ExitCode(cmd.Execute()))
}
