package reporters

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/errs"
)

type PixelReporter struct{}

func NewPixelReporter() *PixelReporter {
	return &PixelReporter{}
}

func (r *PixelReporter) ID() string {
	return "PixelReporter"
}

func (r *PixelReporter) Event(category, action, source, label string, d *dimensions.Values) error {
	pixelURL, err := url.Parse(analytics.AnalyticsURL)
	if err != nil {
		return errs.Wrap(err, "Invalid pixel URL: %s", analytics.AnalyticsURL)
	}

	query := &url.Values{}
	query.Add("x-category", category)
	query.Add("x-action", action)
	query.Add("x-source", source)
	query.Add("x-label", label)

	for num, value := range legacyDimensionMap(d) {
		key := fmt.Sprintf("x-custom%s", num)
		query.Add(key, value)
	}
	pixelURL.RawQuery = query.Encode()

	// logging.Debug("Using S3 pixel URL: %v", pixelURL.String())
	_, err = http.Head(pixelURL.String())
	if err != nil {
		return errs.Wrap(err, "Could not download S3 pixel")
	}

	return nil
}
