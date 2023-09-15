package request

type AnalyticsEvent struct {
	category       string
	action         string
	source         string
	label          string
	dimensionsJson string
	outputType     string
	userID         string
}

func NewAnalyticsEvent(category, action, source, label, dimensionsJson string) *AnalyticsEvent {
	return &AnalyticsEvent{
		category:       category,
		action:         action,
		source:         source,
		label:          label,
		dimensionsJson: dimensionsJson,
	}
}

func (e *AnalyticsEvent) Query() string {
	return `query($category: String!, $action: String!, $source: String!, $label: String, $dimensionsJson: String!) {
		analyticsEvent(category: $category, action: $action, source: $source, label: $label, dimensionsJson: $dimensionsJson) {
			sent
		}
	}`
}

func (e *AnalyticsEvent) Vars() map[string]interface{} {
	return map[string]interface{}{
		"category":       e.category,
		"action":         e.action,
		"source":         e.source,
		"label":          e.label,
		"dimensionsJson": e.dimensionsJson,
	}
}
