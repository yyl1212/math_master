package buildmeta

import (
	"regexp"
	"runtime/debug"
)

// Revision is set only by the release build's linker. Request data never supplies it.
var Revision string
var revisionPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)

func CodeSHA() string {
	if revisionPattern.MatchString(Revision) {
		return Revision
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	sha := ""
	dirty := false
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			sha = s.Value
		}
		if s.Key == "vcs.modified" && s.Value == "true" {
			dirty = true
		}
	}
	if !dirty && revisionPattern.MatchString(sha) {
		return sha
	}
	return ""
}
