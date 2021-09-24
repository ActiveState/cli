package request

type AnalyticsEvent struct {
	category         string
	action           string
	label            string
	projectNameSpace string
	outputType       string
	userID           string
}

func NewAnalyticsEvent(category, action, label, projectNameSpace, outputType, userID string) *AnalyticsEvent {
	return &AnalyticsEvent{
		category:         category,
		action:           action,
		label:            label,
		projectNameSpace: projectNameSpace,
		outputType:       outputType,
		userID:           userID,
	}
}

func (e *AnalyticsEvent) Query() string {
	return `query($category: String!, $action: String!, $label: String, $prjNameSpace: String, $out: String, $userID: String) {
		analyticsEvent(category: $category, action: $action, label: $label, projectNameSpace: $prjNameSpace, output: $out, userID: $userID) {
			sent
		}
	}`
}

func (e *AnalyticsEvent) Vars() map[string]interface{} {
	return map[string]interface{}{
		"category":     e.category,
		"action":       e.action,
		"label":        e.label,
		"prjNameSpace": e.projectNameSpace,
		"out":          e.outputType,
		"userID":       e.userID,
	}
}
