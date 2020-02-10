package request

import "github.com/go-openapi/strfmt"

// OrganizationsByIDs returns the query for retrieving orgs by ids
func OrganizationsByIDs(orgIDs []strfmt.UUID) *organizationByIDs {
	return &organizationByIDs{map[string]interface{}{
		"organization_ids": orgIDs,
	}}
}

type organizationByIDs struct {
	vars map[string]interface{}
}

func (p *organizationByIDs) Query() string {
	return `query ($organization_ids: [uuid!]) {
		organizations(where: {organization_id:{_in: $organization_ids}}) {
		  organization_id
		  display_name
		  url_name
		}
	  }
	  `
}

func (p *organizationByIDs) Vars() map[string]interface{} {
	return p.vars
}
