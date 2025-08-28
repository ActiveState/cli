package reporters

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"

	configMediator "github.com/ActiveState/cli/internal/mediators/config"
)

type PixelReporter struct {
	url string
}

func NewPixelReporter(cfg *config.Instance) *PixelReporter {
	reporter := &PixelReporter{
		url: sourcePixelURL(cfg),
	}

	configMediator.AddListener(constants.AnalyticsPixelOverrideConfig, func() {
		reporter.url = sourcePixelURL(cfg)
	})
	return reporter
}

func (r *PixelReporter) ID() string {
	return "PixelReporter"
}

func (r *PixelReporter) Event(category, action, source, label string, d *dimensions.Values) error {
	pixelURL, err := url.Parse(r.url)
	if err != nil {
		return errs.Wrap(err, "Invalid pixel URL: %s", r.url)
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

func sourcePixelURL(cfg *config.Instance) string {
	var (
		pixelUrl string

		envUrl = os.Getenv(constants.AnalyticsPixelOverrideEnv)
		cfgUrl = cfg.GetString(constants.AnalyticsPixelOverrideConfig)
	)

	switch {
	case envUrl != "":
		pixelUrl = envUrl
	case cfgUrl != "":
		pixelUrl = cfgUrl
	default:
		pixelUrl = constants.DefaultAnalyticsPixel
	}

	return pixelUrl
}
