// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"strings"

	"github.com/otfabric/sparkplugbctl/internal/output"
)

// Process exit codes: the organisation table (docs/exitcodes.md).
const (
	ExitOK           = 0
	ExitUsage        = 2
	ExitInvalidInput = 3
	ExitTransport    = 4
	ExitTimeout      = 5
	ExitPermission   = 9
	ExitInternal     = 10
	ExitInterrupted  = 130 // 128 + SIGINT, as a shell reports it
)

// exitError gives an error chain an exit code. Interruption, timeouts and a
// refused permission in the chain take precedence, see ExitCode.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }

// withExit wraps err so that ExitCode returns code. A nil err stays nil.
func withExit(code int, err error) error {
	if err == nil {
		return nil
	}
	return &exitError{code: code, err: err}
}

// commandEntered records that the command line was accepted: cobra's errors
// before that are usage errors. The root command's pre-run sets it.
var commandEntered bool

// ExitCode maps the error of a run to the process exit code.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	if !commandEntered || strings.HasPrefix(err.Error(), "required flag") {
		return ExitUsage
	}
	var ne net.Error
	switch {
	case errors.Is(err, context.Canceled):
		return ExitInterrupted
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &ne) && ne.Timeout():
		return ExitTimeout
	case errors.Is(err, fs.ErrPermission):
		return ExitPermission
	}
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.code
	}
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return ExitInvalidInput
	}
	if errors.As(err, &ne) {
		return ExitTransport
	}
	return ExitInternal
}

// applyFormatEnv sets --json, when not given, from SPARKPLUGBCTL_FORMAT
// (canonical) or SPARKPLUGBCTL_OUTPUT_FORMAT.
func applyFormatEnv(jsonGiven bool) error {
	v := os.Getenv(envPrefix + "FORMAT")
	if v == "" {
		v = os.Getenv(envPrefix + "OUTPUT_FORMAT")
	}
	if v == "" {
		return nil
	}
	if v != "text" && v != "json" {
		return withExit(ExitUsage, errors.New("invalid output format "+v+": use text or json"))
	}
	if !jsonGiven {
		jsonOut = v == "json"
	}
	return nil
}

// outputError gives a failed write of a command's results (stdout) exit 10,
// "the output could not be written" (docs/exitcodes.md), whatever its cause.
// A nil err stays nil.
func outputError(err error) error {
	if err == nil {
		return nil
	}
	var we *output.WriteError
	if !errors.As(err, &we) {
		err = &output.WriteError{Err: err}
	}
	return withExit(ExitInternal, err)
}

// eventWriter writes the events of host and app to stdout from the
// library's callbacks. The first write that fails is sent on failed, so that
// the command ends with exit 10 instead of running on without output.
type eventWriter struct {
	w      io.Writer
	failed chan error
}

func newEventWriter(w io.Writer) *eventWriter {
	return &eventWriter{w: w, failed: make(chan error, 1)}
}

func (e *eventWriter) printf(format string, args ...any) {
	e.report(fmt.Fprintf(e.w, format, args...))
}

func (e *eventWriter) report(_ int, err error) {
	if err != nil {
		select {
		case e.failed <- err:
		default:
		}
	}
}
