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
		        sources {
		          name
				  version
		          vulnerabilities {
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

type vulnerabilitiesByCommit struct {
	vars map[string]interface{}
}

// VulnerabilitiesByCommit returns the query for retrieving vulnerabilities for a specific commit
func VulnerabilitiesByCommit(commitID string) *vulnerabilitiesByCommit {
	return &vulnerabilitiesByCommit{map[string]interface{}{
		"commit_id": commitID,
	}}
}

func (p *vulnerabilitiesByCommit) Query() string {
	return `query Vulnerabilities($commit_id: Uuid!)
		{
		  commit(commit_id: $commit_id) {
		    __typename
		    ... on Commit {
		      commit_id
		      vulnerability_histogram {
		        severity
		        count
		      }
		      sources {
		        name
				version
		        vulnerabilities {
		          severity
		          cve_id
		          alt_ids
		        }
		      }
		    }
		    ... on NotFound {
		      message
		    }
		  }
		}`
}

func (p *vulnerabilitiesByCommit) Vars() map[string]interface{} {
	return p.vars
}
