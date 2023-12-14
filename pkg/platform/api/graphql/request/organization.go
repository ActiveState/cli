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

// OrganizationsByName returns the query for retrieving org by name
func OrganizationsByName(name string) *organizationByName {
	return &organizationByName{map[string]interface{}{
		"name": name,
	}}
}

type organizationByName struct {
	vars map[string]interface{}
}

func (p *organizationByName) Query() string {
	return `query ($name: String) {
		organizations(where: {url_name:{_eq: $name}}) {
			organization_id
			display_name
			url_name
		}
	}`
}

func (p *organizationByName) Vars() map[string]interface{} {
	return p.vars
}
