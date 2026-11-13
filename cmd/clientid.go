// SPDX-License-Identifier: MIT

package cmd

import (
	"crypto/rand"
	"encoding/hex"
)

// randomClientID returns the MQTT client ID that app, listen and publish use
// without --client-id: sparkplugbctl-<command>- and 8 random hex digits, as
// go-sparkplugb's application default (sparkplug_app_ and 8 hex digits). A
// broker disconnects a client when another connects with its ID
// [MQTT-3.1.4-2], so two runs against one broker must not draw the same one:
// the fixed IDs of listen and publish, used before, made two runs disconnect
// each other, and the clock modulo 100000 of app gave a pair 1 chance in
// 100000. host and edge keep IDs derived from their Sparkplug identity, which
// is unique on its own.
func randomClientID(command string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b) // crypto/rand.Read never returns an error
	return "sparkplugbctl-" + command + "-" + hex.EncodeToString(b)
}
