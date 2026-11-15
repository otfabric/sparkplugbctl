// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"math"

	"github.com/spf13/cobra"
)

// defaultMaxPayload is the largest payload, in bytes, that listen and decode
// decode, unless --max-payload sets another bound: 16 MiB, far above what a
// Sparkplug payload carries in practice. MQTT allows messages up to 256 MiB;
// the broker's own maximum applies before this one.
const defaultMaxPayload = 16 << 20

// maxPayload is --max-payload of listen and decode.
var maxPayload uint64

// addMaxPayloadFlag gives c the --max-payload flag.
func addMaxPayloadFlag(c *cobra.Command) {
	c.Flags().Uint64Var(&maxPayload, "max-payload", defaultMaxPayload, "largest payload to decode, in bytes")
	completeFlags(c, cobra.NoFileCompletions, "max-payload")
}

// payloadLimit returns --max-payload as a length: at least 1, and at most
// math.MaxInt - 1, the largest that a read of one byte more can hold on any
// platform.
func payloadLimit() (int, error) {
	if maxPayload == 0 {
		return 0, withExit(ExitInvalidInput, errors.New("--max-payload must be at least 1"))
	}
	if maxPayload > math.MaxInt-1 {
		return math.MaxInt - 1, nil
	}
	return int(maxPayload), nil
}
