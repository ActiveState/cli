package request

import (
	"time"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
)

func ImpactReport(organization, project string, beforeExpr, afterExpr []byte, beforeTime, afterTime *time.Time) *impactReport {
	var beforeTimeString, afterTimeString *string
	if beforeTime != nil {
		beforeTimeString = ptr.To(beforeTime.Format(time.RFC3339))
	}
	if afterTime != nil {
		afterTimeString = ptr.To(afterTime.Format(time.RFC3339))
	}

	bp := &impactReport{map[string]interface{}{
		"organization": organization,
		"project":      project,
		"beforeExpr":   string(beforeExpr),
		"afterExpr":    string(afterExpr),
		"beforeTime":   beforeTimeString,
		"afterTime":    afterTimeString,
	}}

	return bp
}

type impactReport struct {
	vars map[string]interface{}
}

func (b *impactReport) Query() string {
	return `
query ($organization: String!, $project: String!, $beforeExpr: BuildExpr!, $afterExpr: BuildExpr!, $beforeTime: DateTime, $afterTime: DateTime) {
  impactReport(
    before: {organization: $organization, project: $project, buildExprOrCommit: {buildExpr: $beforeExpr, atTime: $beforeTime}}
    after: {organization: $organization, project: $project, buildExprOrCommit: {buildExpr: $afterExpr, atTime: $afterTime}}
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
