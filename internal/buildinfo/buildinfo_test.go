// SPDX-License-Identifier: MIT

package buildinfo

import (
	"runtime"
	"runtime/debug"
	"testing"
)

const sha = "1802d0f9afe8642d1e278120b76fcdfac967cb9e"

var platform = runtime.Version() + ", " + runtime.GOOS + "/" + runtime.GOARCH

func fakeBuildInfo(t *testing.T, bi *debug.BuildInfo, ok bool) {
	t.Helper()
	old := readBuildInfo
	t.Cleanup(func() { readBuildInfo = old })
	readBuildInfo = func() (*debug.BuildInfo, bool) { return bi, ok }
}

func TestTextAllFields(t *testing.T) {
	fakeBuildInfo(t, nil, false)
	got := Resolve("sparkplugbctl", "0.4.0", "v0.4.0", sha, "2026-11-05T15:41:49Z").String()
	want := "sparkplugbctl 0.4.0 (tag v0.4.0, commit " + sha + ", built 2026-11-05T15:41:49Z, " + platform + ")"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestTextWithoutTag(t *testing.T) {
	fakeBuildInfo(t, nil, false)
	got := Resolve("sparkplugbctl", "v0.4.0-3-gabc", "", sha, "2026-11-05T15:41:49Z").String()
	want := "sparkplugbctl 0.4.0-3-gabc (commit " + sha + ", built 2026-11-05T15:41:49Z, " + platform + ")"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestDevFallbackNothingKnown(t *testing.T) {
	fakeBuildInfo(t, nil, false)
	got := Resolve("sparkplugbctl", "", "", "", "").String()
	want := "sparkplugbctl dev (commit unknown, built unknown, " + platform + ")"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestDevFallbackFromVCS(t *testing.T) {
	fakeBuildInfo(t, &debug.BuildInfo{
		Main: debug.Module{Version: "(devel)"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: sha},
			{Key: "vcs.modified", Value: "true"},
		},
	}, true)
	got := Resolve("sparkplugbctl", "", "", "", "").String()
	want := "sparkplugbctl dev (commit " + sha + "-dirty, built unknown, " + platform + ")"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestGoInstallVersion(t *testing.T) {
	fakeBuildInfo(t, &debug.BuildInfo{
		Main:     debug.Module{Version: "v0.4.0"},
		Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: sha}, {Key: "vcs.modified", Value: "false"}},
	}, true)
	i := Resolve("sparkplugbctl", "", "", "", "")
	if i.Version != "0.4.0" || i.Commit != sha || i.Tag != "" {
		t.Fatalf("got %+v", i)
	}
}

func TestInjectedWinsOverBuildInfo(t *testing.T) {
	fakeBuildInfo(t, &debug.BuildInfo{
		Main:     debug.Module{Version: "v9.9.9"},
		Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "other"}},
	}, true)
	i := Resolve("sparkplugbctl", "0.4.0", "v0.4.0", sha, "x")
	if i.Version != "0.4.0" || i.Commit != sha {
		t.Fatalf("got %+v", i)
	}
}

func TestJSONKeyOrderAndValues(t *testing.T) {
	fakeBuildInfo(t, nil, false)
	got := Resolve("sparkplugbctl", "0.4.0", "v0.4.0", sha, "2026-11-05T15:41:49Z").JSON()
	want := `{"name":"sparkplugbctl","version":"0.4.0","tag":"v0.4.0","commit":"` + sha +
		`","built":"2026-11-05T15:41:49Z","go":"` + runtime.Version() + `","platform":"` + runtime.GOOS + "/" + runtime.GOARCH + `"}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestJSONDev(t *testing.T) {
	fakeBuildInfo(t, nil, false)
	got := Resolve("sparkplugbctl", "", "", "", "").JSON()
	want := `{"name":"sparkplugbctl","version":"dev","tag":"","commit":"unknown","built":"unknown","go":"` +
		runtime.Version() + `","platform":"` + runtime.GOOS + "/" + runtime.GOARCH + `"}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestPseudoAndDirtyVersionsAreDev(t *testing.T) {
	for _, v := range []string{
		"(devel)", "",
		"v0.3.7-0.20261107212211-a54bc2e3e6f9",
		"v0.3.7-0.20261107212211-a54bc2e3e6f9+dirty",
		"v0.0.0-20261107212211-a54bc2e3e6f9",
		"v0.4.0-rc.1.0.20261107212211-a54bc2e3e6f9",
		"v0.4.0+dirty",
	} {
		fakeBuildInfo(t, &debug.BuildInfo{Main: debug.Module{Version: v}}, true)
		if got := Resolve("sparkplugbctl", "", "", "", "").Version; got != "dev" {
			t.Errorf("%q: version %q, want dev", v, got)
		}
	}
	fakeBuildInfo(t, &debug.BuildInfo{Main: debug.Module{Version: "v0.4.0-rc.1"}}, true)
	if got := Resolve("sparkplugbctl", "", "", "", "").Version; got != "0.4.0-rc.1" {
		t.Errorf("v0.4.0-rc.1: version %q", got)
	}
}
