package request

import "fmt"

func Build(owner, project, commitID string) *buildPlanByBuild {
	return &buildPlanByBuild{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"commitID":     commitID,
	}}
}

type buildPlanByBuild struct {
	vars map[string]interface{}
}

func (b *buildPlanByBuild) Query() string {
	return fmt.Sprintf(`
mutation ($organization: String!, $project: String!, $commitID: String!) {
  build(organization: $organization, project: $project, vcsRef: $commitID) {
    %s
  }
}
	`, buildResultFragment)
}

func (b *buildPlanByBuild) Vars() map[string]interface{} {
	return b.vars
}
