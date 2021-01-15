package request

// VulnerabilitiesByProject returns the query for retrieving vulnerabilities by projects
func VulnerabilitiesByProject(org, name string) *vulnerabilitiesByProject {
	return &vulnerabilitiesByProject{map[string]interface{}{
		"org":  org,
		"name": name,
	}}
}

type vulnerabilitiesByProject struct {
	vars map[string]interface{}
}

func (p *vulnerabilitiesByProject) Query() string {
	return `query Vulnerabilities($org: String!, $name: String!)
		{
		  project(org: $org, name: $name) {
		    __typename
		    ... on Project {
		      name
		      description
		      commit {
		        commit_id
		        vulnerability_histogram {
		          severity
		          count
		        }
		        ingredients {
		          name
		          vulnerabilities {
		            ingredient_version
		            severity
		            cve_id
		            alt_ids
		          }
		        }
		      }
		    }
		    ... on NotFound {
		      message
		    }
		  }
		}`
}

func (p *vulnerabilitiesByProject) Vars() map[string]interface{} {
	return p.vars
}
