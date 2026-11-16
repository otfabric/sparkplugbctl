// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/otfabric/go-sparkplugb/mqtt"
)

// recTransport is an mqtt.Transport without a broker that records every
// call and keeps the handlers of the subscriptions, so that a test can
// deliver messages and look at what was published at a given moment.
// Publishing to a topic that contains failTopic fails with publishErr; a
// Connect is recorded and returns connectErr.
type recTransport struct {
	failTopic  string
	publishErr error
	connectErr error

	mu        sync.Mutex
	calls     []string // "Connect", "Subscribe <topic>", "Publish <topic>", "Disconnect"
	published map[string][][]byte
	handlers  map[string]mqtt.MessageHandler
	connect   mqtt.ConnectOpts
}

func (r *recTransport) record(call string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, call)
}

func (r *recTransport) Connect(_ context.Context, o mqtt.ConnectOpts) error {
	r.mu.Lock()
	r.connect = o
	r.mu.Unlock()
	r.record("Connect")
	return r.connectErr
}

func (r *recTransport) Publish(_ context.Context, topic string, _ byte, _ bool, data []byte) error {
	if r.publishErr != nil && strings.Contains(topic, r.failTopic) {
		return r.publishErr
	}
	r.mu.Lock()
	if r.published == nil {
		r.published = map[string][][]byte{}
	}
	r.published[topic] = append(r.published[topic], append([]byte(nil), data...))
	r.mu.Unlock()
	r.record("Publish " + topic)
	return nil
}

func (r *recTransport) Subscribe(_ context.Context, topic string, _ byte, h mqtt.MessageHandler) error {
	r.mu.Lock()
	if r.handlers == nil {
		r.handlers = map[string]mqtt.MessageHandler{}
	}
	r.handlers[topic] = h
	r.mu.Unlock()
	r.record("Subscribe " + topic)
	return nil
}

func (r *recTransport) Disconnect(context.Context) error {
	r.record("Disconnect")
	return nil
}

// callList returns the calls so far.
func (r *recTransport) callList() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

// count returns how many calls start with prefix.
func (r *recTransport) count(prefix string) int {
	n := 0
	for _, c := range r.callList() {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

// deliver hands a message on topic to the handler subscribed to filter.
func (r *recTransport) deliver(t *testing.T, filter, topic string, data []byte) {
	t.Helper()
	r.mu.Lock()
	h := r.handlers[filter]
	r.mu.Unlock()
	if h == nil {
		t.Fatalf("no subscription to %s; calls %v", filter, r.callList())
	}
	h(topic, data)
}

// manualTicker replaces the publish interval of edge with ticks the test
// sends; tick returns once edge has received the tick.
type manualTicker struct{ c chan time.Time }

func injectTicker(t *testing.T) *manualTicker {
	t.Helper()
	m := &manualTicker{c: make(chan time.Time)}
	old := newTicker
	newTicker = func(time.Duration) (<-chan time.Time, func()) { return m.c, func() {} }
	t.Cleanup(func() { newTicker = old })
	return m
}

func (m *manualTicker) tick(t *testing.T) {
	t.Helper()
	select {
	case m.c <- time.Now():
	case <-time.After(10 * time.Second):
		t.Fatal("edge did not take the tick")
	}
}
