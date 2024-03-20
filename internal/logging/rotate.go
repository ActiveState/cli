package logging

import (
	"io/fs"
	"io/ioutil" //nolint:staticcheck
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

func rotateLogs(files []fs.FileInfo, timeCutoff time.Time, amountCutoff int) []fs.FileInfo {
	rotate := []fs.FileInfo{}

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
			if currentPrefix == prefix && strings.HasSuffix(file.Name(), FileNameSuffix) {
				c = c + 1
				if c > amountCutoff && file.ModTime().Before(timeCutoff) {
					rotate = append(rotate, file)
				}
			}
		}
	}

	return rotate
}

func rotateLogsOnDisk() {
	// Clean up old log files
	logDir := filepath.Dir(FilePath())
	files, err := ioutil.ReadDir(logDir) //nolint:staticcheck
	if err != nil && !os.IsNotExist(err) {
		Error("Could not scan config dir to clean up stale logs: %v", err)
		return
	}

	// Prevent running over this logic too often as it affects performance
	// https://activestatef.atlassian.net/browse/DX-1516
	if len(files) < 30 {
		return
	}

	rotate := rotateLogs(files, time.Now().Add(-time.Hour), 10)
	for _, file := range rotate {
		if err := os.Remove(filepath.Join(logDir, file.Name())); err != nil {
			Error("Could not clean up old log: %s, error: %v", file.Name(), err)
		}
	}
}

var stopTimer chan bool

// StartRotateLogTimer starts log rotation on a timer and returns a function that should be called to stop it.
func StartRotateLogTimer() func() {
	interval := 1 * time.Minute
	if durationString := os.Getenv(constants.SvcLogRotateIntervalEnvVarName); durationString != "" {
		if duration, err := strconv.Atoi(durationString); err == nil {
			interval = time.Duration(duration) * time.Millisecond
		}
	}

	stopTimer = make(chan bool)
	go func() {
		rotateLogsOnDisk()
		for {
			select {
			case <-stopTimer:
				return
			case <-time.After(interval):
				rotateLogsOnDisk()
			}
		}
	}()

	return func() {
		stopTimer <- true
	}
}
