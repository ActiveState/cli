package logging

import (
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"time"
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
