package analytics

import "github.com/ActiveState/cli/internal/analytics/dimensions"

// Dispatcher describes a struct that can send analytics event in the background
type Dispatcher interface {
	Event(category string, action string, dim ...*dimensions.Values)
	EventWithLabel(category string, action string, label string, dim ...*dimensions.Values)
	Wait()
	Close()
}
