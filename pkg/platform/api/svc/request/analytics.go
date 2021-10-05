package request

type AnalyticsEvent struct {
	category         string
	action           string
	label            string
	projectNameSpace string
	outputType       string
}

func NewAnalyticsEvent(category, action, label, projectNameSpace, outputType string) *AnalyticsEvent {
	return &AnalyticsEvent{
		category:         category,
		action:           action,
		label:            label,
		projectNameSpace: projectNameSpace,
		outputType:       outputType,
	}
}

func (e *AnalyticsEvent) Query() string {
	return `query($category: String!, $action: String!, $label: String, $prjNameSpace: String, $out: String) {
		analyticsEvent(category: $category, action: $action, label: $label, projectNameSpace: $prjNameSpace, output: $out) {
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
	}
}

type AuthenticationEvent struct {
	userID string
}

func NewAuthenticationEvent(userID string) *AuthenticationEvent {
	return &AuthenticationEvent{
		userID: userID,
	}
}

func (e *AuthenticationEvent) Query() string {
	return `query($userID: String!) {
		authenticationEvent(userID: $userID) {
			dummy
		}
	}`
}

func (e *AuthenticationEvent) Vars() map[string]interface{} {
	return map[string]interface{}{
		"userID": e.userID,
	}
}
