package reporters

import (
	"encoding/json"
	"path/filepath"

	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
)

type TestReporter struct {
	path string
	cfg  *config.Instance
}

const TestReportFilename = "analytics.log"

func TestReportFilepath() string {
	appdata := storage.AppDataPath()
	logging.Warning("Appdata path: %s", appdata)
	return filepath.Join(appdata, TestReportFilename)
}

func NewTestReporter(path string, cfg *config.Instance) *TestReporter {
	reporter := &TestReporter{path, cfg}
	configMediator.AddListener(constants.AnalyticsPixelOverrideConfig, func() {
		reporter.cfg = cfg
	})
	return reporter
}

func (r *TestReporter) ID() string {
	return "TestReporter"
}

type TestLogEntry struct {
	Category   string
	Action     string
	Source     string
	Label      string
	URL        string
	Dimensions *dimensions.Values
}

func (r *TestReporter) Event(category, action, source, label string, d *dimensions.Values) error {
	url := r.cfg.GetString(constants.AnalyticsPixelOverrideConfig)
	b, err := json.Marshal(TestLogEntry{category, action, source, label, url, d})
	if err != nil {
		return errs.Wrap(err, "Could not marshal test log entry")
	}
	b = append(b, []byte("\n\x00")...)

	if err := fileutils.AmendFileLocked(r.path, b, fileutils.AmendByAppend); err != nil {
		return errs.Wrap(err, "Could not write to file")
	}
	return nil
}
