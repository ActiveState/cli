package rollrest

import "github.com/davidji99/simpleresty"

// NotificationsService handles communication with the notification related
// methods of the Rollbar API.
//
// Rollbar API docs: https://explorer.docs.rollbar.com/#tag/Notifications
type NotificationsService service

// PDIntegrationRequest represents a request to configure Rollbar with PagerDuty.
type PDIntegrationRequest struct {
	Enabled    bool   `json:"enabled"`
	ServiceKey string `json:"service_key,omitempty"`
}

// PDRuleRequest represents a request to add one or many PagerDuty notification rule.
type PDRuleRequest struct {
	// As of Feb. 11th 2020, the only possible value for the Triggers field is `new_item`.
	Trigger string          `json:"trigger,omitempty"`
	Filters []*PDRuleFilter `json:"filters,omitempty"`
	Config  *PDRuleConfig   `json:"config,omitempty"`
}

// PDRuleFilter represents a PagerDuty rule filter.
type PDRuleFilter struct {
	Type      string `json:"type,omitempty"`
	Operation string `json:"operation,omitempty"`
	Value     string `json:"value,omitempty"`
	Path      string `json:"path,omitempty"`
	Period    int    `json:"period,omitempty"`
	Count     int    `json:"count,omitempty"`
}

// PDRuleConfig represents the configuration options available on a rule.
type PDRuleConfig struct {
	// PagerDuty Service API Key. Make sure the ServiceKey string value is length 32.
	ServiceKey string `json:"service_key,omitempty"`
}

// ConfigurePagerDutyIntegration configures the PagerDuty integration for a project.
//
// This function creates and modifies the PagerDuty integration. Requires a project access token.
//
// Rollbar API docs: https://explorer.docs.rollbar.com/#operation/configuring-pagerduty-integration
func (n *NotificationsService) ConfigurePagerDutyIntegration(opts *PDIntegrationRequest) (*simpleresty.Response, error) {
	urlStr := n.client.http.RequestURL("/notifications/pagerduty")

	// Set the correct authentication header
	n.client.setAuthTokenHeader(n.client.projectAccessToken)

	// Execute the request
	response, getErr := n.client.http.Put(urlStr, nil, opts)

	return response, getErr
}

// ModifyPagerDutyRules creates & modifies PagerDuty notification rules for a project.
//
// Requires a project access token.
// (The API documentation is wrong regarding which access token to use as of Feb. 10th, 2020.)
//
// Additionally, if you construct a request body that has an empty array for filters or is missing entirely,
// a default rule is created: 'trigger in any environment where level >= debug'.
//
// Rollbar API docs: https://explorer.docs.rollbar.com/#operation/setup-pagerduty-notification-rules
func (n *NotificationsService) ModifyPagerDutyRules(opts []*PDRuleRequest) (bool, *simpleresty.Response, error) {
	urlStr := n.client.http.RequestURL("/notifications/pagerduty/rules")

	// Set the correct authentication header
	n.client.setAuthTokenHeader(n.client.projectAccessToken)

	// Execute the request
	response, getErr := n.client.http.Put(urlStr, nil, opts)
	if getErr != nil {
		return false, response, getErr
	}

	return true, response, nil
}

// DeleteAllPagerDutyRules removes all rules for a project's PagerDuty notification integration.
// This is the same ModifyPagerDutyRules but passes in an empty array as the request body for convenience.
//
// Requires a project access token.
// (The API documentation is wrong regarding which access token to use as of Feb. 10th, 2020.)
//
// Rollbar API docs: https://explorer.docs.rollbar.com/#operation/setup-pagerduty-notification-rules
func (n *NotificationsService) DeleteAllPagerDutyRules() (bool, *simpleresty.Response, error) {
	opts := make([]*PDRuleRequest, 0)
	return n.ModifyPagerDutyRules(opts)
}
