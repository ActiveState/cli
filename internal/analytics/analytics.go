package analytics

import (
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/constants"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
)

// Dispatcher describes a struct that can send analytics event in the background
type Dispatcher interface {
	Event(category string, action string, dim ...*dimensions.Values)
	EventWithLabel(category string, action string, label string, dim ...*dimensions.Values)
	Wait()
	Close()
}

func init() {
	configMediator.RegisterOption(constants.ReportAnalyticsConfig, configMediator.Bool, configMediator.EmptyEvent, configMediator.EmptyEvent)
}
