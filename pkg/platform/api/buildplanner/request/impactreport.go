package request

func ImpactReport(organization, project string, beforeExpr, afterExpr []byte) *impactReport {
	bp := &impactReport{map[string]interface{}{
		"organization": organization,
		"project":      project,
		"beforeExpr":   string(beforeExpr),
		"afterExpr":    string(afterExpr),
	}}

	return bp
}

type impactReport struct {
	vars map[string]interface{}
}

func (b *impactReport) Query() string {
	return `
query ($organization: String!, $project: String!, $beforeExpr: BuildExpr!, $afterExpr: BuildExpr!) {
  impactReport(
    before: {organization: $organization, project: $project, buildExprOrCommit: {buildExpr: $beforeExpr}}
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
      buildBefore {
        __typename
      }
      buildAfter {
        __typename
      }
      message
    }
  }
}
`
}

func (b *impactReport) Vars() (map[string]interface{}, error) {
	return b.vars, nil
}
