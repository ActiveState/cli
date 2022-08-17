package request

func ProjectByOrgAndName(org string, project string) *projectByOrgAndName {
	return &projectByOrgAndName{map[string]interface{}{
		"org":  org,
		"name": project,
	}}
}

type projectByOrgAndName struct {
	vars map[string]interface{}
}

func (p *projectByOrgAndName) Query() string {
	return `query ($org: String, $name: String) {
	  projects(where: {deleted: {_is_null: true}, name: {_ilike: $name}, organization: {url_name: {_ilike: $org}}}, limit: 1) {
		branches {
		  branch_id
		  commit_id
		  main
		  project_id
		  tracking_type
		  tracks
		  label
		}
		description
		name
		added
		created_by
		forked_from
		forked_project {
		  name
		  organization {
			url_name
		  }
		}
		changed
		managed
		organization_id
		private
		project_id
		repo_url
	  }
	}
	`
}

func (p *projectByOrgAndName) Vars() map[string]interface{} {
	return p.vars
}
