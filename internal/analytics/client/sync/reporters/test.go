package reporters

import (
	"encoding/json"
	"path/filepath"

	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
)

type TestReporter struct {
	path string
}

const TestReportFilename = "analytics.log"

func TestReportFilepath() string {
	appdata, err := storage.AppDataPath()
	if err != nil {
		logging.Warning("Could not acquire appdata path, using cwd instead. Error received: %s", errs.JoinMessage(err))
	}
	return filepath.Join(appdata, TestReportFilename)
}

func NewTestReporter(path string) *TestReporter {
	return &TestReporter{path}
}

func (r *TestReporter) ID() string {
	return "TestReporter"
}

type TestLogEntry struct {
	Category   string
	Action     string
	Source     string
	Label      string
	Dimensions *dimensions.Values
}

func (r *TestReporter) Event(category, action, source, label string, d *dimensions.Values) error {
	b, err := json.Marshal(TestLogEntry{category, action, source, label, d})
	if err != nil {
		return errs.Wrap(err, "Could not marshal test log entry")
	}
	b = append(b, []byte("\n\x00")...)

	if err := fileutils.AmendFileLocked(r.path, b, fileutils.AmendByAppend); err != nil {
		return errs.Wrap(err, "Could not write to file")
	}
	return nil
}
