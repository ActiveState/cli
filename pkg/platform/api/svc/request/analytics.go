package request

type AnalyticsEvent struct {
	category         string
	action           string
	label            string
	dimensionsJson  string
	outputType       string
	userID           string
}

func NewAnalyticsEvent(category, action, label, dimensionsJson string) *AnalyticsEvent {
	return &AnalyticsEvent{
		category:         category,
		action:           action,
		label:            label,
		dimensionsJson:   dimensionsJson,
	}
}

func (e *AnalyticsEvent) Query() string {
	return `query($category: String!, $action: String!, $label: String, $dimensionsJson: String!) {
		analyticsEvent(category: $category, action: $action, label: $label, dimensionsJson: $dimensionsJson) {
			sent
		}
	}`
}

func (e *AnalyticsEvent) Vars() map[string]interface{} {
	return map[string]interface{}{
		"category":     e.category,
		"action":       e.action,
		"label":        e.label,
		"dimensionsJson": e.dimensionsJson,
	}
}
