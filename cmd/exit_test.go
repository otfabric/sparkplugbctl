// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"testing"
)

func TestExitCode(t *testing.T) {
	commandEntered = true
	t.Cleanup(func() { commandEntered = false })
	for _, c := range []struct {
		err  error
		want int
	}{
		{nil, ExitOK},
		{errors.New("required flag(s) \"topic\" not set"), ExitUsage},
		{fmt.Errorf("x: %w", context.Canceled), ExitInterrupted},
		{fmt.Errorf("x: %w", context.DeadlineExceeded), ExitTimeout},
		{withExit(ExitTransport, fmt.Errorf("x: %w", context.DeadlineExceeded)), ExitTimeout},
		{withExit(ExitInvalidInput, fs.ErrPermission), ExitPermission},
		{withExit(ExitTransport, errors.New("connect: refused")), ExitTransport},
		{&fs.PathError{Op: "open", Path: "x", Err: fs.ErrNotExist}, ExitInvalidInput},
		{&net.OpError{Op: "dial", Err: errors.New("refused")}, ExitTransport},
		{errors.New("boom"), ExitInternal},
		{withExit(ExitTransport, nil), ExitOK},
	} {
		if got := ExitCode(c.err); got != c.want {
			t.Errorf("ExitCode(%v) = %d, want %d", c.err, got, c.want)
		}
	}
}

func TestCommandExitCodes(t *testing.T) {
	for _, c := range []struct {
		args []string
		want int
	}{
		{[]string{"no-such-command"}, ExitUsage},
		{[]string{"publish"}, ExitUsage},
		{[]string{"decode", "--file", "/does/not/exist"}, ExitInvalidInput},
		{[]string{"encode", "--metric", "bad"}, ExitInvalidInput},
	} {
		commandEntered = false
		_, _, err := run(t, nil, c.args...)
		if got := ExitCode(err); got != c.want {
			t.Errorf("%v: exit %d, want %d (%v)", c.args, got, c.want, err)
		}
	}
}

func TestFormatEnv(t *testing.T) {
	t.Setenv("SPARKPLUGBCTL_FORMAT", "")
	t.Setenv("SPARKPLUGBCTL_OUTPUT_FORMAT", "json")
	if _, _, err := run(t, nil, "version"); err != nil || !jsonOut {
		t.Errorf("OUTPUT_FORMAT ignored: %v", err)
	}
	t.Setenv("SPARKPLUGBCTL_FORMAT", "text")
	if _, _, err := run(t, nil, "version"); err != nil || jsonOut {
		t.Errorf("FORMAT does not win: %v", err)
	}
	t.Setenv("SPARKPLUGBCTL_FORMAT", "xml")
	commandEntered = false
	_, _, err := run(t, nil, "version")
	if got := ExitCode(err); got != ExitUsage {
		t.Errorf("invalid format: exit %d", got)
	}
}
