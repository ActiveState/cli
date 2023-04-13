package logging

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
)

var LogPrefixRx = regexp.MustCompile(`^[a-zA-Z\-]+`)

func rotateLogs() {
	// Clean up old log files
	logDir := filepath.Dir(FilePath())
	files, err := ioutil.ReadDir(logDir)
	if err != nil && !os.IsNotExist(err) {
		Error("Could not scan config dir to clean up stale logs: %v", err)
		return
	}

	// Prevent running over this logic too often as it affects performance
	// https://activestatef.atlassian.net/browse/DX-1516
	if len(files) < 30 {
		return
	}

	timeCutoff := time.Now().Add(-time.Hour)
	amountCutoff := 10

	Debug("Rotating logs with time cutoff of: %v", timeCutoff)

	sort.Slice(files, func(i, j int) bool { return files[i].ModTime().After(files[j].ModTime()) })

	// Collect the possible file prefixes that we're going to want to run through
	prefixes := map[string]struct{}{}
	for _, file := range files {
		prefix := LogPrefixRx.FindString(file.Name())
		if _, exists := prefixes[prefix]; !exists {
			prefixes[prefix] = struct{}{}
		}
	}

	for prefix := range prefixes {
		c := 0
		for _, file := range files {
			currentPrefix := LogPrefixRx.FindString(file.Name())
			if currentPrefix != prefix || !strings.HasSuffix(file.Name(), FileNameSuffix) {
				continue
			}
			c = c + 1
			if c <= amountCutoff || file.ModTime().After(timeCutoff) {
				continue
			}
			err := os.Remove(filepath.Join(logDir, file.Name()))
			if err != nil {
				Error("Could not clean up old log: %s, error: %v", file.Name(), err)
			}
		}
		if c > amountCutoff {
			Debug("Cleaned up %d old log files", c-amountCutoff)
		}
	}
}

// RotateLogListener rotates logs on a timer.
// Run this as a Go routine.
func RotateLogTimer() {
	timeout := 1 * time.Minute
	if durationString := os.Getenv(constants.SvcLogRotateTimerDurationEnvVarName); durationString != "" {
		if duration, err := strconv.Atoi(durationString); err == nil {
			timeout = time.Duration(duration) * time.Second
		}
	}

	rotateLogs()
	for {
		tick := time.Tick(timeout)
		select {
		case <-tick:
			rotateLogs()
		}
	}
}
