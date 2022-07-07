package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

const (
	DevelopmentVersion = "devel"
)

var (
	// Version holds the complete version number. Filled in at linking time.
	Version = DevelopmentVersion

	// Revision is filled with the VCS (e.g. git) revision being used to build
	// the program at linking time.
	Revision = ""
)

func Short() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Sprintf("dagger %s (%s)", DevelopmentVersion, "")
	}
	settings := info.Settings

	fmt.Println(settings)
	Revision, ok = getSetting(settings, "vcs.revision")
	if !ok {
		return fmt.Sprintf("dagger %s (%s)", DevelopmentVersion, "")
	}
	// we take the short revision hash
	Revision = Revision[:8]

	modified, ok := getSetting(settings, "vcs.modified")
	if !ok {
		return fmt.Sprintf("dagger %s (%s)", DevelopmentVersion, "")
	}

	if modified == "true" {
		Version = DevelopmentVersion
	}

	return fmt.Sprintf("dagger %s (%s)", Version, Revision)
}

func getSetting(settings []debug.BuildSetting, key string) (string, bool) {
	for _, s := range settings {
		if s.Key == key {
			return s.Value, true
		}
	}
	return "", false
}

func Long() string {
	return fmt.Sprintf("%s %s/%s", Short(), runtime.GOOS, runtime.GOARCH)
}
