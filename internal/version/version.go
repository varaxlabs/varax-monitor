package version

import (
	"fmt"
	"runtime"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func String() string {
	return fmt.Sprintf("onax %s (commit: %s, built: %s, go: %s)",
		Version, Commit, Date, runtime.Version())
}
