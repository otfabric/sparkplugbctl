// SPDX-License-Identifier: MIT

package cmd

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// complete runs the hidden __complete command and returns the offered
// values and the directive line.
func complete(t *testing.T, args ...string) ([]string, string) {
	t.Helper()
	out, _, err := run(t, nil, append([]string{cobra.ShellCompRequestCmd}, args...)...)
	removeCommands(cobra.ShellCompRequestCmd) // cobra keeps it once called
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	directive := lines[len(lines)-1]
	var vals []string
	for _, l := range lines[:len(lines)-1] {
		v, _, _ := strings.Cut(l, "\t")
		vals = append(vals, v)
	}
	return vals, directive
}

// removeCommands removes the commands named name that cobra adds to rootCmd
// on its first run and keeps: the completion command writes to the output
// of the run that created it, and __complete stays once called.
func removeCommands(name string) {
	for _, c := range rootCmd.Commands() {
		if c.Name() == name {
			rootCmd.RemoveCommand(c)
		}
	}
}

func TestCompletionShells(t *testing.T) {
	got, _ := complete(t, "completion", "")
	for _, sh := range []string{"bash", "zsh", "fish", "powershell"} {
		if !slices.Contains(got, sh) {
			t.Errorf("completion offers %v, missing %s", got, sh)
		}
	}
	for _, sh := range []string{"bash", "zsh", "fish", "powershell"} {
		removeCommands("completion")
		out, _, err := run(t, nil, "completion", sh)
		if err != nil || !strings.Contains(out, "sparkplugbctl") {
			t.Errorf("completion %s: %v, %d bytes", sh, err, len(out))
		}
	}
}

func TestCompleteMetricTypes(t *testing.T) {
	for _, c := range []string{"publish", "edge", "encode"} {
		got, directive := complete(t, c, "--metric", "temp:")
		if len(got) != len(metricTypes) || got[0] != "temp:Int8=" || !slices.Contains(got, "temp:Double=") {
			t.Errorf("%s --metric temp: offers %v", c, got)
		}
		want := cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		if directive != ":"+strconv.Itoa(int(want)) {
			t.Errorf("%s --metric directive %s, want :%d", c, directive, want)
		}
	}
	if got, _ := complete(t, "publish", "--metric", "t:do"); !slices.Equal(got, []string{"t:Double="}) {
		t.Errorf("t:do offers %v", got)
	}
	for _, typed := range []string{"temp", "temp:Double=2"} {
		if got, directive := complete(t, "publish", "--metric", typed); len(got) != 0 || directive != ":4" {
			t.Errorf("%q offers %v %s", typed, got, directive)
		}
	}
}

func TestCompleteFileFlags(t *testing.T) {
	got, directive := complete(t, "listen", "--tls-key", "")
	if !slices.Equal(got, []string{"pem", "key"}) || directive != ":8" {
		t.Errorf("--tls-key offers %v %s", got, directive)
	}
	got, directive = complete(t, "app", "--tls-ca", "")
	if !slices.Equal(got, []string{"pem", "crt", "cer"}) || directive != ":8" {
		t.Errorf("--tls-ca offers %v %s", got, directive)
	}
	if _, directive := complete(t, "decode", "--file", ""); directive != ":0" {
		t.Errorf("decode --file directive %s, want :0 (files)", directive)
	}
	if _, directive := complete(t, "host", "--host-id", ""); directive != ":4" {
		t.Errorf("host --host-id directive %s, want :4 (no files)", directive)
	}
	if _, directive := complete(t, "version", ""); directive != ":4" {
		t.Errorf("version args directive %s, want :4 (no files)", directive)
	}
}

// TestEveryFlagCompletes walks the command tree: every flag that takes a
// value has a completion function, and every command without subcommands
// completes no files for its (absent) arguments.
func TestEveryFlagCompletes(t *testing.T) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		check := func(f *pflag.Flag) {
			if f.NoOptDefVal != "" || f.Name == "help" {
				return
			}
			if _, ok := c.GetFlagCompletionFunc(f.Name); !ok {
				t.Errorf("%s --%s has no completion function", c.CommandPath(), f.Name)
			}
		}
		c.LocalFlags().VisitAll(check)
		c.InheritedFlags().VisitAll(check)
		if c.Name() != "completion" && c.Name() != "help" && !c.HasSubCommands() && c.ValidArgsFunction == nil {
			t.Errorf("%s has no ValidArgsFunction", c.CommandPath())
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	rootCmd.InitDefaultHelpCmd()
	rootCmd.InitDefaultCompletionCmd()
	walk(rootCmd)
}
