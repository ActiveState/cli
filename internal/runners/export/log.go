package export

import (
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
)

type Log struct {
	output.Outputer
}

func NewLog(prime primeable) *Log {
	return &Log{prime.Output()}
}

type LogParams struct {
	Prefix string
	Index  int
}

type logFile struct {
	Name      string
	Timestamp int
}

var ErrInvalidLogIndex = errs.New("invalid index")
var ErrInvalidLogPrefix = errs.New("invalid log prefix")
var ErrLogNotFound = errs.New("log not found")

func (l *Log) Run(params *LogParams) (rerr error) {
	defer rationalizeError(&rerr)

	if params.Index < 0 {
		return ErrInvalidLogIndex
	}
	if params.Prefix == "" {
		params.Prefix = "state"
	}

	// Fetch list of log files.
	logDir := filepath.Dir(logging.FilePath())
	logFiles, err := fileutils.ListDirSimple(logDir, false)
	if err != nil {
		return errs.Wrap(err, "failed to list log files")
	}

	// Filter down the list based on the given prefix.
	filteredLogFiles := []*logFile{}
	regex, err := regexp.Compile(params.Prefix + `-\d+-(\d+)\.log`)
	if err != nil {
		return ErrInvalidLogPrefix
	}
	for _, file := range logFiles {
		if regex.MatchString(file) {
			timestamp, err := strconv.Atoi(regex.FindStringSubmatch(file)[1])
			if err != nil {
				continue
			}
			filteredLogFiles = append(filteredLogFiles, &logFile{file, timestamp})
		}
	}

	// Sort logs in ascending order by name (which include timestamp), not modification time.
	sort.SliceStable(filteredLogFiles, func(i, j int) bool {
		return filteredLogFiles[i].Timestamp > filteredLogFiles[j].Timestamp // sort ascending, not descending
	})

	if params.Index >= len(filteredLogFiles) {
		return ErrLogNotFound
	}

	l.Outputer.Print(output.Prepare(
		filteredLogFiles[params.Index].Name,
		&struct {
			LogFile string `json:"logFile"`
		}{filteredLogFiles[params.Index].Name},
	))

	return nil
}
