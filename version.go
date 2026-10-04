package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"
)

// repoURL is where this program's source lives.
const repoURL = "https://github.com/samtcifihi/obstacle-gen-playground"

// sourceVersion returns the git commit the server was built from, and
// whether the source had changes that weren't committed, or "" if it can't
// tell. go build records these in the binary, but go run (which the
// justfile uses) doesn't, so then it asks git about the directory this file
// was compiled from.
var sourceVersion = sync.OnceValues(func() (commit string, modified bool) {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				commit = s.Value
			case "vcs.modified":
				modified = s.Value == "true"
			}
		}
		if commit != "" {
			return commit, modified
		}
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", false
	}
	dir := filepath.Dir(file)
	head, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", false
	}
	status, err := exec.Command("git", "-C", dir, "status", "--porcelain").Output()
	return string(bytes.TrimSpace(head)), err == nil && len(bytes.TrimSpace(status)) > 0
})
