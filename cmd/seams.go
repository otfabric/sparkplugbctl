// SPDX-License-Identifier: MIT

package cmd

import (
	"time"

	"github.com/otfabric/go-sparkplugb/mqtt"
	"github.com/otfabric/go-sparkplugb/payload"
	"github.com/otfabric/sparkplugbctl/internal/broker"
)

// newTransport builds the MQTT transport of a command. Tests replace it to
// inject transport faults that a real broker cannot produce on demand.
var newTransport = func(o broker.Opts) (mqtt.Transport, error) {
	t, err := broker.NewTransport(o)
	if err != nil {
		return nil, err // not a nil *PahoTransport in a non-nil interface
	}
	return t, nil
}

// encodePayload marshals a payload. Metrics that the command line can build
// always marshal, so tests replace it to reach the failure path.
var encodePayload = payload.Encode

// newTicker starts the publish interval of edge and returns its channel and
// the function that stops it. Tests replace it to tick on demand.
var newTicker = func(d time.Duration) (<-chan time.Time, func()) {
	t := time.NewTicker(d)
	return t.C, t.Stop
}
