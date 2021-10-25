package reporters

import (
	"fmt"
	"net/url"

	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/logging"
)

type LegacyPixelReporter struct {}

func NewLegacyPixelReporter() *LegacyPixelReporter {
	return &LegacyPixelReporter{}
}

func (r *LegacyPixelReporter) ID() string {
	return "LegacyPixelReporter"
}

func (r *LegacyPixelReporter) Event(category, action, label string, d *dimensions.Values) error {
	query := &url.Values{}
	query.Add("x-category", category)
	query.Add("x-action", action)
	query.Add("x-label", label)

	for num, value := range legacyDimensionMap(d) {
		key := fmt.Sprintf("x-custom%s", num)
		query.Add(key, value)
	}
	fullQuery := query.Encode()

	logging.Debug("Using S3 pixel query: %v", fullQuery)
	svcExec := appinfo.SvcApp().Exec()
	_, err := exeutils.ExecuteAndForget(svcExec, []string{"_event", query.Encode()})
	if err != nil {
		return errs.Wrap(err, "Failed to send legacy event to state-svc")
	}

	return nil
}
