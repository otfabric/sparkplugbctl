// SPDX-License-Identifier: MIT

package main

import (
	"os"

	"github.com/otfabric/sparkplugbctl/cmd"
)

// Injected at link time (Makefile and the release workflow):
// -X main.version=... -X main.tag=... -X main.commit=... -X main.date=...
// Empty means not injected; see internal/buildinfo for the fallbacks.
var (
	version string
	tag     string
	commit  string
	date    string
)

// exit ends the process; a test replaces it.
var exit = os.Exit

func main() {
	cmd.Version, cmd.Tag, cmd.Commit, cmd.BuildDate = version, tag, commit, date
	exit(cmd.ExitCode(cmd.Execute()))
}
