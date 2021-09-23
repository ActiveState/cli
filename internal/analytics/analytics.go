package analytics

import (
	"github.com/ActiveState/cli/internal/logging"
)

// AnalyticsDispatcher describes a struct that can send analytics event in the background
type AnalyticsDispatcher interface {
	Event(category string, action string)
	EventWithLabel(category string, action string, label string)
	Wait()
}

func handlePanics(err interface{}, stack []byte) {
	if err == nil {
		return
	}
	logging.Error("Panic in client analytics: %v", err)
	logging.Debug("Stack: %s", string(stack))
}
