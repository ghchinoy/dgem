package cmd

import (
	"fmt"
	"runtime/debug"
)

// Build metadata, set at build time with
//
//	-ldflags "-X github.com/ghchinoy/dgem/cmd.Version=v0.1.0 -X github.com/ghchinoy/dgem/cmd.Commit=<sha> -X github.com/ghchinoy/dgem/cmd.Date=<rfc3339>"
//
// Version is `git describe --tags --match 'v*'`: a release tag such as v0.1.0 on a release build, or
// v0.1.0-3-gabc1234 for commits after it. Unset values fall back to Go's embedded VCS info.
var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

func buildRevision() string {
	if Commit != "" {
		return Commit
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			if s.Key == "vcs.revision" && len(s.Value) >= 7 {
				return s.Value[:7]
			}
		}
	}
	return "unknown"
}

// VersionInfo is reported by `dgem --version` and the gateway's /health.
func VersionInfo() map[string]string {
	return map[string]string{"version": Version, "revision": buildRevision(), "build_date": Date}
}

func versionString() string {
	s := fmt.Sprintf("%s (revision %s", Version, buildRevision())
	if Date != "" {
		s += ", built " + Date
	}
	return s + ")"
}

func init() {
	RootCmd.Version = versionString()
}
