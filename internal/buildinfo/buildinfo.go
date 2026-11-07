// SPDX-License-Identifier: MIT

// Package buildinfo resolves the build metadata of the binary and formats the
// output of the version command: one text line or one JSON object.
//
// The values come from the linker (-X main.version=... -X main.tag=...
// -X main.commit=... -X main.date=..., set by the Makefile and the release
// workflow). What was not injected falls back to the Go build information
// (go install ...@vX.Y.Z, or the VCS stamp of go build), then to "dev" or
// "unknown".
package buildinfo

import (
	"encoding/json"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
)

// readBuildInfo is debug.ReadBuildInfo; tests replace it.
var readBuildInfo = debug.ReadBuildInfo

// Info is the resolved build metadata. Field order is the JSON key order.
type Info struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Tag      string `json:"tag"`
	Commit   string `json:"commit"`
	Built    string `json:"built"`
	Go       string `json:"go"`
	Platform string `json:"platform"`
}

// Resolve builds the Info of the binary name from the injected values, any of
// which may be empty.
func Resolve(name, version, tag, commit, built string) Info {
	bi, ok := readBuildInfo()
	if version == "" && ok && isRelease(bi.Main.Version) {
		version = bi.Main.Version
	}
	version = strings.TrimPrefix(version, "v")
	if version == "" {
		version = "dev"
	}
	if commit == "" && ok {
		commit = vcsCommit(bi)
	}
	if commit == "" {
		commit = "unknown"
	}
	if built == "" {
		built = "unknown"
	}
	return Info{
		Name:     name,
		Version:  version,
		Tag:      tag,
		Commit:   commit,
		Built:    built,
		Go:       runtime.Version(),
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
	}
}

// pseudoVersion matches the timestamp-and-commit tail of a Go pseudo-version
// (v0.3.7-0.20261107212211-a54bc2e3e6f9), which go build (Go 1.24+) stamps on
// a build from a checkout.
var pseudoVersion = regexp.MustCompile(`[-.]\d{14}-[0-9a-f]{12}$`)

// isRelease reports whether v, the main module version of the build
// information, is a released version (go install ...@vX.Y.Z), not "(devel)",
// a pseudo-version or a +dirty build from modified sources.
func isRelease(v string) bool {
	return v != "" && v != "(devel)" && !strings.Contains(v, "+") && !pseudoVersion.MatchString(v)
}

// vcsCommit is the vcs.revision of the build, with -dirty when vcs.modified.
func vcsCommit(bi *debug.BuildInfo) string {
	var rev string
	var dirty bool
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev != "" && dirty {
		rev += "-dirty"
	}
	return rev
}

// String is the version line:
// <name> <version> (tag <tag>, commit <commit>, built <built>, <go>, <goos>/<goarch>);
// the tag part is left out when there is no tag.
func (i Info) String() string {
	var b strings.Builder
	b.WriteString(i.Name + " " + i.Version + " (")
	if i.Tag != "" {
		b.WriteString("tag " + i.Tag + ", ")
	}
	b.WriteString("commit " + i.Commit + ", built " + i.Built + ", " + i.Go + ", " + i.Platform + ")")
	return b.String()
}

// JSON is the version object on one line, keys in a fixed order.
func (i Info) JSON() string {
	b, _ := json.Marshal(i) // a struct of strings always marshals
	return string(b)
}
