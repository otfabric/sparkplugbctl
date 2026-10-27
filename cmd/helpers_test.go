// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	mochi "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func startBroker(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	srv := mochi.New(&mochi.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err := srv.AddHook(new(auth.AllowHook), nil); err != nil {
		t.Fatal(err)
	}
	if err := srv.AddListener(listeners.NewTCP(listeners.Config{ID: "test", Address: addr})); err != nil {
		t.Fatal(err)
	}
	if err := srv.Serve(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	return "tcp://" + addr
}

func resetFlags(c *cobra.Command) {
	reset := func(f *pflag.Flag) {
		if sv, ok := f.Value.(pflag.SliceValue); ok {
			_ = sv.Replace(nil)
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	}
	c.Flags().VisitAll(reset)
	c.PersistentFlags().VisitAll(reset)
	for _, sub := range c.Commands() {
		resetFlags(sub)
	}
}

type result struct {
	stdout *syncBuffer
	stderr *syncBuffer
	done   chan error
}

func (r *result) wait(t *testing.T) error {
	t.Helper()
	select {
	case err := <-r.done:
		return err
	case <-time.After(15 * time.Second):
		t.Fatal("command did not return")
		return nil
	}
}

func start(ctx context.Context, stdin io.Reader, args ...string) *result {
	resetFlags(rootCmd)
	r := &result{stdout: &syncBuffer{}, stderr: &syncBuffer{}, done: make(chan error, 1)}
	rootCmd.SetArgs(args)
	rootCmd.SetOut(r.stdout)
	rootCmd.SetErr(r.stderr)
	if stdin == nil {
		stdin = strings.NewReader("")
	}
	rootCmd.SetIn(stdin)
	for _, sub := range rootCmd.Commands() {
		sub.SetContext(ctx)
	}
	go func() { r.done <- rootCmd.ExecuteContext(ctx) }()
	return r
}

func run(t *testing.T, stdin io.Reader, args ...string) (string, string, error) {
	t.Helper()
	r := start(context.Background(), stdin, args...)
	err := r.wait(t)
	return r.stdout.String(), r.stderr.String(), err
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func contains(b *syncBuffer, s string) func() bool {
	return func() bool { return strings.Contains(b.String(), s) }
}
