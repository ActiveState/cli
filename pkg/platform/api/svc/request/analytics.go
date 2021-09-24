package request

type AnalyticsEvent struct {
	category    string
	action      string
	label       string
	projectName string
	outputType  string
	userID      string
}

func NewAnalyticsEvent(category, action, label, projectName, outputType, userID string) *AnalyticsEvent {
	return &AnalyticsEvent{
		category:    category,
		action:      action,
		label:       label,
		projectName: projectName,
		outputType:  outputType,
		userID:      userID,
	}
}

func (e *AnalyticsEvent) Query() string {
	return `query($c: String!, $a: String!, $l: String, $pn: String, $out: String, $uid: String) {
		analyticsEvent(category: $c, action: $a, label: $l, project: $pn, output: $out, userID: $uid) {
			sent
		}
	}`
}

func (e *AnalyticsEvent) Vars() map[string]interface{} {
	return map[string]interface{}{
		"c":   e.category,
		"a":   e.action,
		"l":   e.label,
		"pn":  e.projectName,
		"out": e.outputType,
	}
}
