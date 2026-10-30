// SPDX-License-Identifier: MIT

package cmd

import (
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
