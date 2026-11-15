// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// networkCommands are the commands that connect, with the stderr line each
// prints once it is connected; publish ends by itself.
var networkCommands = map[string]struct {
	args  []string
	ready string
}{
	"listen":  {[]string{"listen"}, "Listening on spBv1.0/#"},
	"app":     {[]string{"app"}, "Listening for edge activity"},
	"host":    {[]string{"host", "--host-id", "h"}, "Listening for edge activity"},
	"edge":    {[]string{"edge", "--group", "g", "--node", "n", "--interval", "1h"}, "Publishing data every"},
	"publish": {[]string{"publish", "--topic", "spBv1.0/g/NDATA/n", "--metric", "a:Int32=1"}, ""},
}

// runNetwork runs a network command until it is connected, then stops it,
// and returns its stdout, its stderr and its error. A command that fails
// before it is connected returns that error.
func runNetwork(t *testing.T, name string, extra ...string) (string, string, error) {
	t.Helper()
	c := networkCommands[name]
	args := append(append([]string(nil), c.args...), extra...)
	if c.ready == "" {
		return run(t, nil, args...)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := start(ctx, nil, args...)
	deadline := time.Now().Add(10 * time.Second)
	for !strings.Contains(r.stderr.String(), c.ready) {
		select {
		case err := <-r.done:
			return r.stdout.String(), r.stderr.String(), err
		case <-time.After(10 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s neither connected nor failed:\n%s", name, r.stderr.String())
		}
	}
	cancel()
	err := r.wait(t)
	return r.stdout.String(), r.stderr.String(), err
}

func writeFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestCredentials runs every network command against a broker that admits
// one login: the username and password reach the broker from the flags,
// the environment and the password file; a wrong password and no login are
// refused by the broker (exit 4). The password is in no output.
func TestCredentials(t *testing.T) {
	const user, pass = "operator", "pw-4711-secret"
	url, hook := startAuthBroker(t, user, pass)
	file := writeFile(t, pass+"\n")
	cases := []struct {
		name  string
		env   map[string]string
		args  []string
		want  int
		login string
	}{
		{"flags", nil, []string{"--username", user, "--password", pass}, ExitOK, user},
		{"environment", map[string]string{"SPARKPLUGBCTL_USERNAME": user, "SPARKPLUGBCTL_PASSWORD": pass}, nil, ExitOK, user},
		{"password file", nil, []string{"--username", user, "--password-file", file}, ExitOK, user},
		{"password file variable", map[string]string{"SPARKPLUGBCTL_USERNAME": user, "SPARKPLUGBCTL_PASSWORD_FILE": file}, nil, ExitOK, user},
		{"file flag over password variable", map[string]string{"SPARKPLUGBCTL_PASSWORD": "not-this-one"}, []string{"--username", user, "--password-file", file}, ExitOK, user},
		{"password flag over file variable", map[string]string{"SPARKPLUGBCTL_PASSWORD_FILE": "/does/not/exist"}, []string{"--username", user, "--password", pass}, ExitOK, user},
		{"wrong password", nil, []string{"--username", user, "--password", "wrong"}, ExitTransport, user},
		{"no login", nil, nil, ExitTransport, ""},
	}
	for name := range networkCommands {
		for _, c := range cases {
			t.Run(name+"/"+c.name, func(t *testing.T) {
				for k, v := range c.env {
					t.Setenv(k, v)
				}
				hook.logins()
				stdout, stderr, err := runNetwork(t, name, append([]string{"--broker", url}, c.args...)...)
				if got := ExitCode(err); got != c.want {
					t.Fatalf("exit %d, want %d (%v)\n%s", got, c.want, err, stderr)
				}
				if logins := hook.logins(); !slices.Contains(logins, c.login) {
					t.Errorf("broker saw logins %q, want %q", logins, c.login)
				}
				if strings.Contains(stdout+stderr, pass) {
					t.Errorf("the password is in the output:\n%s\n%s", stdout, stderr)
				}
			})
		}
	}
}

// TestPasswordWithoutUsername checks that a password without a username,
// which MQTT cannot carry, is not sent and is reported on stderr, by every
// network command alike: the broker sees an anonymous CONNECT.
func TestPasswordWithoutUsername(t *testing.T) {
	url, hook := startAuthBroker(t, "operator", "pw")
	open := startBroker(t)
	for name := range networkCommands {
		t.Run(name, func(t *testing.T) {
			hook.logins()
			_, stderr, err := runNetwork(t, name, "--broker", url, "--password", "pw")
			if got := ExitCode(err); got != ExitTransport {
				t.Errorf("exit %d, want %d (%v)", got, ExitTransport, err)
			}
			if !strings.Contains(stderr, "warning: the password is not sent without --username") {
				t.Errorf("stderr = %q", stderr)
			}
			// After a refusal the MQTT client tries once more with MQTT 3.1.
			if logins := hook.logins(); len(logins) == 0 || slices.ContainsFunc(logins, func(u string) bool { return u != "" }) {
				t.Errorf("broker saw logins %q, want anonymous ones only", logins)
			}
			if _, _, err := runNetwork(t, name, "--broker", open, "--password", "pw"); err != nil {
				t.Errorf("against a broker that admits anonymous clients: %v", err)
			}
		})
	}
}

// TestPasswordFile checks --password-file and SPARKPLUGBCTL_PASSWORD_FILE:
// one trailing line end is not part of the password, a file that cannot be
// read or holds no password is invalid input (exit 3), and the password
// given twice is a usage error (exit 2).
func TestPasswordFile(t *testing.T) {
	const user, pass = "operator", "pw-4711-secret"
	url, _ := startAuthBroker(t, user, pass)
	login := []string{"publish", "--broker", url, "--topic", "spBv1.0/g/NDATA/n", "--username", user}
	for _, c := range []struct {
		name    string
		env     map[string]string
		args    []string
		want    int
		message string
	}{
		{"newline", nil, []string{"--password-file", writeFile(t, pass+"\n")}, ExitOK, ""},
		{"crlf", nil, []string{"--password-file", writeFile(t, pass+"\r\n")}, ExitOK, ""},
		{"no line end", nil, []string{"--password-file", writeFile(t, pass)}, ExitOK, ""},
		{"two line ends", nil, []string{"--password-file", writeFile(t, pass+"\n\n")}, ExitTransport, ""},
		{"missing file", nil, []string{"--password-file", filepath.Join(t.TempDir(), "missing")}, ExitInvalidInput, "password file"},
		{"empty file", nil, []string{"--password-file", writeFile(t, "\n")}, ExitInvalidInput, "is empty"},
		{"too long", nil, []string{"--password-file", writeFile(t, strings.Repeat("x", maxPasswordBytes+1))}, ExitInvalidInput, "longer than the 65535 bytes"},
		{"flag and file", nil, []string{"--password", pass, "--password-file", writeFile(t, pass)}, ExitUsage, "both set"},
		{"both variables", map[string]string{"SPARKPLUGBCTL_PASSWORD": pass, "SPARKPLUGBCTL_PASSWORD_FILE": writeFile(t, pass)}, nil, ExitUsage, "both set"},
	} {
		t.Run(c.name, func(t *testing.T) {
			for k, v := range c.env {
				t.Setenv(k, v)
			}
			stdout, stderr, err := run(t, nil, append(append([]string(nil), login...), c.args...)...)
			if got := ExitCode(err); got != c.want {
				t.Fatalf("exit %d, want %d (%v)", got, c.want, err)
			}
			if c.message != "" && (err == nil || !strings.Contains(err.Error(), c.message)) {
				t.Errorf("error %v, want it to say %q", err, c.message)
			}
			if strings.Contains(stdout+stderr, pass) || (err != nil && strings.Contains(err.Error(), pass)) {
				t.Errorf("the password is in the output")
			}
		})
	}

	// A command that does not connect does not read the file.
	t.Setenv("SPARKPLUGBCTL_PASSWORD_FILE", filepath.Join(t.TempDir(), "missing"))
	if _, _, err := run(t, nil, "version"); err != nil {
		t.Errorf("version with a missing password file: %v", err)
	}
}

func TestReadPasswordBounds(t *testing.T) {
	for _, c := range []struct {
		name    string
		content string
		want    string
		ok      bool
	}{
		{"one byte", "x", "x", true},
		{"max-1", strings.Repeat("x", maxPasswordBytes-1), strings.Repeat("x", maxPasswordBytes-1), true},
		{"max", strings.Repeat("x", maxPasswordBytes), strings.Repeat("x", maxPasswordBytes), true},
		{"max and crlf", strings.Repeat("x", maxPasswordBytes) + "\r\n", strings.Repeat("x", maxPasswordBytes), true},
		{"max+1", strings.Repeat("x", maxPasswordBytes+1), "", false},
		{"much longer", strings.Repeat("x", 1<<20), "", false},
		{"spaces kept", " pw ", " pw ", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := readPassword(writeFile(t, c.content))
			if (err == nil) != c.ok || got != c.want {
				t.Errorf("readPassword = %d bytes, %v; want %d bytes, ok=%t", len(got), err, len(c.want), c.ok)
			}
		})
	}
}

// TestPasswordHelp checks that the help marks --password deprecated, names
// the alternatives, and still lists it.
func TestPasswordHelp(t *testing.T) {
	out, _, err := run(t, nil, "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"--password string        MQTT password (deprecated: use --password-file or SPARKPLUGBCTL_PASSWORD)",
		"--password-file string   file holding the MQTT password",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("help lacks %q:\n%s", want, out)
		}
	}
}
