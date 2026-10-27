// SPDX-License-Identifier: MIT

package cmd

import "testing"

func TestRootCommandExists(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}
	if rootCmd.Use == "" {
		t.Error("rootCmd.Use is empty")
	}
}
