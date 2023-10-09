package util

import (
	"fmt"
	"runtime/debug"
	"time"
)

func VersionFromBuildInfo() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "n/a"
	}
	var (
		rev   string
		mod   time.Time
		dirty string
	)
	for _, setting := range buildInfo.Settings {
		switch setting.Key {
		case "vcs.time":
			var err error
			mod, err = time.Parse(time.RFC3339, setting.Value)
			if err != nil {
				mod = time.Time{}
			}
		case "vcs.revision":
			rev = setting.Value[:7]
		case "vcs.modified":
			if setting.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	return fmt.Sprintf("%s-%s%s", mod.Format("20060102150405"), rev, dirty)
}
