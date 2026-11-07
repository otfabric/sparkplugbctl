// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/otfabric/go-sparkplugb/payload"
)

func TestRootCommandExists(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}
	if rootCmd.Use == "" {
		t.Error("rootCmd.Use is empty")
	}
}

func TestVersion(t *testing.T) {
	defer func(v, tg, c, d string) { Version, Tag, Commit, BuildDate = v, tg, c, d }(Version, Tag, Commit, BuildDate)
	Version, Tag, Commit, BuildDate = "0.4.0", "v0.4.0", "1802d0f9afe8642d1e278120b76fcdfac967cb9e", "2026-11-05T15:41:49Z"
	platform := runtime.Version() + ", " + runtime.GOOS + "/" + runtime.GOARCH
	wantText := "sparkplugbctl 0.4.0 (tag v0.4.0, commit 1802d0f9afe8642d1e278120b76fcdfac967cb9e, built 2026-11-05T15:41:49Z, " + platform + ")\n"
	wantJSON := `{"name":"sparkplugbctl","version":"0.4.0","tag":"v0.4.0","commit":"1802d0f9afe8642d1e278120b76fcdfac967cb9e",` +
		`"built":"2026-11-05T15:41:49Z","go":"` + runtime.Version() + `","platform":"` + runtime.GOOS + "/" + runtime.GOARCH + `"}` + "\n"

	t.Setenv("SPARKPLUGBCTL_FORMAT", "")
	t.Setenv("SPARKPLUGBCTL_OUTPUT_FORMAT", "")
	cases := []struct {
		name, format string
		args         []string
		want         string
	}{
		{"text", "", []string{"version"}, wantText},
		{"json flag", "", []string{"version", "--json"}, wantJSON},
		{"format json", "json", []string{"version"}, wantJSON},
		{"format text", "text", []string{"version"}, wantText},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("SPARKPLUGBCTL_FORMAT", c.format)
			out, _, err := run(t, nil, c.args...)
			if err != nil {
				t.Fatal(err)
			}
			if out != c.want {
				t.Errorf("got  %q\nwant %q", out, c.want)
			}
		})
	}
}

// TestEnvironment checks that a global flag comes from its SPARKPLUGBCTL_
// variable, that the command line takes precedence, and that a bad value
// is an error naming the variable.
func TestEnvironment(t *testing.T) {
	t.Setenv("SPARKPLUGBCTL_JSON", "true")
	if _, _, err := run(t, nil, "encode", "--metric", "x:Int32=1", "--file", filepath.Join(t.TempDir(), "p.bin")); err != nil {
		t.Fatal(err)
	}
	if !jsonOut {
		t.Error("SPARKPLUGBCTL_JSON=true did not set --json")
	}

	if _, _, err := run(t, nil, "encode", "--json=false", "--metric", "x:Int32=1", "--file", filepath.Join(t.TempDir(), "p.bin")); err != nil {
		t.Fatal(err)
	}
	if jsonOut {
		t.Error("--json=false on the command line did not take precedence")
	}

	t.Setenv("SPARKPLUGBCTL_JSON", "maybe")
	if _, _, err := run(t, nil, "version"); err == nil || !strings.Contains(err.Error(), "SPARKPLUGBCTL_JSON") {
		t.Errorf("err = %v, want an error naming SPARKPLUGBCTL_JSON", err)
	}
}

func TestExecute(t *testing.T) {
	resetFlags(rootCmd)
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs([]string{"version"})
	if err := Execute(); err != nil {
		t.Fatal(err)
	}
	rootCmd.SetArgs([]string{"no-such-command"})
	if err := Execute(); err == nil {
		t.Fatal("want error for unknown command")
	}
}

func TestEncodeDecodeStdio(t *testing.T) {
	out, _, err := run(t, nil, "encode", "--seq", "7", "--metric", "temp:Double=21.5", "--metric", "ok:Boolean=true")
	if err != nil {
		t.Fatal(err)
	}
	p, err := payload.Decode([]byte(out))
	if err != nil {
		t.Fatal(err)
	}
	if p.GetSeq() != 7 || len(p.GetMetrics()) != 2 {
		t.Fatalf("decoded seq=%d metrics=%d", p.GetSeq(), len(p.GetMetrics()))
	}

	text, _, err := run(t, strings.NewReader(out), "decode")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "seq=7") || !strings.Contains(text, "temp") || !strings.Contains(text, "21.5") {
		t.Fatalf("decode text output:\n%s", text)
	}

	js, _, err := run(t, strings.NewReader(out), "decode", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(js, `"seq": 7`) || !strings.Contains(js, `"name": "ok"`) {
		t.Fatalf("decode json output:\n%s", js)
	}
}

func TestEncodeDecodeFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.bin")
	_, stderr, err := run(t, nil, "encode", "--file", path, "--metric", "a:Int32=3")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, "Wrote") || !strings.Contains(stderr, path) {
		t.Fatalf("stderr = %q", stderr)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	out, _, err := run(t, nil, "decode", "--file", path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "a") || !strings.Contains(out, "Int32") {
		t.Fatalf("decode output:\n%s", out)
	}
}

func TestEncodeErrors(t *testing.T) {
	if _, _, err := run(t, nil, "encode", "--metric", "bad"); err == nil {
		t.Error("want error for bad metric")
	}
	missingDir := filepath.Join(t.TempDir(), "missing", "p.bin")
	if _, _, err := run(t, nil, "encode", "--file", missingDir, "--metric", "a:Int32=1"); err == nil || !strings.Contains(err.Error(), "write") {
		t.Errorf("want write error, got %v", err)
	}
}

func TestDecodeErrors(t *testing.T) {
	if _, _, err := run(t, nil, "decode", "--file", filepath.Join(t.TempDir(), "nope")); err == nil || !strings.Contains(err.Error(), "open") {
		t.Errorf("want open error, got %v", err)
	}
	if _, _, err := run(t, strings.NewReader("\xff\xff\xff"), "decode"); err == nil || !strings.Contains(err.Error(), "decode protobuf") {
		t.Errorf("want decode error, got %v", err)
	}
	if _, _, err := run(t, iotest.ErrReader(errors.New("boom")), "decode"); err == nil || !strings.Contains(err.Error(), "read input") {
		t.Errorf("want read error, got %v", err)
	}
}
