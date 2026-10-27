// SPDX-License-Identifier: MIT

package main

import (
	"os"

	"github.com/otfabric/sparkplugbctl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
