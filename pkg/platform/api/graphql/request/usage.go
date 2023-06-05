package request

import (
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/platform/api/graphql/model"
)

// RuntimeUsage reports rtusage for the last 8 days, ensuring we reach into the previous week regardless of timezone.
// The use-case of this function is to get the current rtusage, but in order to do this we have to request the last 78
// days rather than the current time, because the week field does not have timezone data, and so requesting the current
// date will result in timezone miss-match issues.
// It is then up to the consuming code to use the most recent rtusage week (ie. the first element in the slice)
func RuntimeUsage(organizationID strfmt.UUID) *usage {
	return &usage{vars: map[string]interface{}{
		"organization_id": organizationID,
		"week":            model.Date{time.Now().Add(-(8 * 24 * time.Hour))},
	}}
}

type usage struct {
	vars map[string]interface{}
}

func (p *usage) Query() string {
	return `
		query ($organization_id: uuid!, $week: date) {
			  organizations_runtime_usage(limit: 1, order_by: [{week_of: desc}], where: {_and: [{organization_id: {_eq: $organization_id}}, {week_of: {_gte: $week}}]}) {
					limit_runtimes
					active_runtimes
			  }
	  }	  
	  `
}

func (p *usage) Vars() (map[string]interface{}, error) {
	return p.vars, nil
}
