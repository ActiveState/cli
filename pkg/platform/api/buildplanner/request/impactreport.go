package request

import (
	"github.com/go-openapi/strfmt"
)

func ImpactReport(organization, project string, beforeCommitId strfmt.UUID, afterExpr []byte) *impactReport {
	bp := &impactReport{map[string]interface{}{
		"organization":   organization,
		"project":        project,
		"beforeCommitId": beforeCommitId.String(),
		"afterExpr":      string(afterExpr),
	}}

	return bp
}

type impactReport struct {
	vars map[string]interface{}
}

func (b *impactReport) Query() string {
	return `
query ($organization: String!, $project: String!, $beforeCommitId: ID!, $afterExpr: BuildExpr!) {
  impactReport(
    before: {organization: $organization, project: $project, buildExprOrCommit: {commitId: $beforeCommitId}}
    after: {organization: $organization, project: $project, buildExprOrCommit: {buildExpr: $afterExpr}}
  ) {
    __typename
    ... on ImpactReport {
      ingredients {
        namespace
        name
        before {
          ingredientID
          version
          isRequirement
        }
        after {
          ingredientID
          version
          isRequirement
        }
      }
    }
    ... on ImpactReportError {
      message
    }
  }
}
`
}

func (b *impactReport) Vars() (map[string]interface{}, error) {
	return b.vars, nil
}
