package test

import (
	"fmt"
	"github.com/davidji99/rollrest-go/rollrest"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestConfigurePagerDutyIntegration(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/notifications/pagerduty", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("generic_response.json"))
	})

	opts := &rollrest.PDIntegrationRequest{
		Enabled:    true,
		ServiceKey: "6xFrzw2uSLtfKvXRdGUYU86XKgqdD9rs",
	}
	r, err := client.Notifications.ConfigurePagerDutyIntegration(opts)

	assert.Nil(t, err)
	assert.Equal(t, "PUT", r.RequestMethod)
	assert.Equal(t, 200, r.StatusCode)
}

func TestModifyPagerDutyRules(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/notifications/pagerduty/rules", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("generic_response.json"))
	})

	opts := []*rollrest.PDRuleRequest{
		{
			Trigger: "new_item",
			Filters: []*rollrest.PDRuleFilter{
				{
					Type:      "level",
					Operation: "gte",
					Value:     "critical",
				},
				{
					Type:      "title",
					Operation: "within",
					Value:     "some_title",
				},
			},
			Config: &rollrest.PDRuleConfig{ServiceKey: "fFGnZhAWunwRc5EaGCHAzR727fDjRW6X"},
		},
		{
			Trigger: "new_item",
			Filters: []*rollrest.PDRuleFilter{
				{
					Type:      "title",
					Operation: "nwithin",
					Value:     "some_freeform_string",
				},
			},
		},
	}

	isCreated, r, err := client.Notifications.ModifyPagerDutyRules(opts)

	assert.Nil(t, err)
	assert.Equal(t, "PUT", r.RequestMethod)
	assert.Equal(t, true, isCreated)
	assert.Equal(t, "[{\"trigger\":\"new_item\",\"filters\":[{\"type\":\"level\",\"operation\":\"gte\",\"value\":\"critical\"},{\"type\":\"title\",\"operation\":\"within\",\"value\":\"some_title\"}],\"config\":{\"service_key\":\"fFGnZhAWunwRc5EaGCHAzR727fDjRW6X\"}},{\"trigger\":\"new_item\",\"filters\":[{\"type\":\"title\",\"operation\":\"nwithin\",\"value\":\"some_freeform_string\"}]}]", r.RequestBody)
}

func TestDeleteAllPagerDutyRules(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/notifications/pagerduty/rules", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("generic_response.json"))
	})

	isDeleted, r, err := client.Notifications.DeleteAllPagerDutyRules()
	assert.Nil(t, err)
	assert.Equal(t, "PUT", r.RequestMethod)
	assert.Equal(t, true, isDeleted)
	assert.Equal(t, "[]", r.RequestBody)
}
