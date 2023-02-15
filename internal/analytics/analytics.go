package analytics

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
)

// Dispatcher describes a struct that can send analytics event in the background
type Dispatcher interface {
	Event(category string, action string, dim ...*Dimensions)
	EventWithLabel(category string, action string, label string, dim ...*Dimensions)
	Wait()
	Close()
}

func init() {
	configMediator.RegisterOption(constants.ReportAnalyticsConfig, configMediator.Bool, configMediator.EmptyEvent, configMediator.EmptyEvent)
}

func CalculateFlags() string {
	flags := []string{}
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
		}
	}
	return strings.Join(flags, " ")
}
