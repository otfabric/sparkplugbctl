// SPDX-License-Identifier: MIT

package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

// Shell completion. Cobra's default completion command serves bash, zsh, fish
// and PowerShell; the functions here give the flags their values.

// metricTypes are the metric types of --metric name:type=value, as
// parseDataType accepts them (case-insensitive), in their Sparkplug spelling.
var metricTypes = []string{
	"Int8", "Int16", "Int32", "Int64",
	"UInt8", "UInt16", "UInt32", "UInt64",
	"Float", "Double", "Boolean", "String", "Text", "UUID", "Bytes",
}

// completeMetric completes the type of a --metric name:type=value: after
// "name:" it offers "name:<type>=" for each type that starts with what was
// typed. Before the colon (a free name) and after "=" (a value) it offers
// nothing.
func completeMetric(_ *cobra.Command, _ []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	name, typed, ok := strings.Cut(toComplete, ":")
	if !ok || strings.Contains(typed, "=") {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var out []cobra.Completion
	for _, t := range metricTypes {
		if strings.HasPrefix(strings.ToLower(t), strings.ToLower(typed)) {
			out = append(out, name+":"+t+"=")
		}
	}
	return out, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}

// anyFile completes file names of any kind (a binary payload has no fixed
// extension).
func anyFile(*cobra.Command, []string, string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveDefault
}

// fileExt completes files with one of the extensions.
func fileExt(exts ...string) cobra.CompletionFunc {
	return func(*cobra.Command, []string, string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return exts, cobra.ShellCompDirectiveFilterFileExt
	}
}

// completeFlags registers f for the flags names of c (local or persistent).
// A registration error is a typo in a flag name; TestEveryFlagCompletes
// catches it.
func completeFlags(c *cobra.Command, f cobra.CompletionFunc, names ...string) {
	for _, n := range names {
		_ = c.RegisterFlagCompletionFunc(n, f)
	}
}
