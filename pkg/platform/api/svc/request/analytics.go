package request

type AnalyticsEvent struct {
	category    string
	action      string
	label       string
	projectName string
	outputType  string
}

func NewAnalyticsEvent(category, action, label, projectName, outputType string) *AnalyticsEvent {
	return &AnalyticsEvent{
		category:    category,
		action:      action,
		label:       label,
		projectName: projectName,
		outputType:  outputType,
	}
}

func (e *AnalyticsEvent) Query() string {
	return `query($c: String!, $a: String!, $l: String, $pn: String, $out: String) {
		analyticsEvent(category: $c, action: $a, label: $l, project: $pn, output: $out) {
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
