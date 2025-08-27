package analytics

import (
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
)

// Dispatcher describes a struct that can send analytics event in the background
type Dispatcher interface {
	Event(category, action string, dim ...*dimensions.Values)
	EventWithLabel(category, action, label string, dim ...*dimensions.Values)
	EventWithSource(category, action, source string, dim ...*dimensions.Values)
	Wait()
	Close()
}

var AnalyticsURL string

func init() {
	configMediator.RegisterOption(constants.ReportAnalyticsConfig, configMediator.Bool, true)
	configMediator.RegisterOption(constants.AnalyticsPixelOverrideConfig, configMediator.String, "")
}

func SetConfig(cfg *config.Instance) {
	AnalyticsURL = cfg.GetString(constants.AnalyticsPixelOverrideConfig)
}

func RegisterConfigListener(cfg *config.Instance) error {
	configMediator.AddListener(constants.AnalyticsPixelOverrideConfig, func() {
		AnalyticsURL = cfg.GetString(constants.AnalyticsPixelOverrideConfig)
	})
	return nil
}
