package analytics

// AnalyticsDispatcher describes a struct that can send analytics event in the background
type AnalyticsDispatcher interface {
	Event(category string, action string)
	EventWithLabel(category string, action string, label string)
	Wait()
	AuthenticationUpdate(userID string)
}
