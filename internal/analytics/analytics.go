package analytics

import (
	"github.com/ActiveState/cli/internal/analytics/dimensions"
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

func init() {
	configMediator.RegisterOption(configMediator.Option{
		Name:    constants.ReportAnalyticsConfig,
		Type:    configMediator.Bool,
		Default: true,
	})
}
