package request

import (
	"time"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
)

func Evaluate(organization, project string, expr []byte, atTime *time.Time, dynamic bool, target string) *evaluate {
	eval := &evaluate{map[string]interface{}{
		"organization": organization,
		"project":      project,
		"expr":         string(expr),
		"target":       target,
	}}

	var timestamp *string
	if atTime != nil {
		timestamp = ptr.To(atTime.Format(time.RFC3339))
	}
	if !dynamic {
		eval.vars["atTime"] = timestamp
	} else {
		eval.vars["atTime"] = "dynamic"
	}

	return eval
}

type evaluate struct {
	vars map[string]interface{}
}

func (b *evaluate) Query() string {
	return `
query ($organization: String!, $project: String!, $expr: BuildExpr!, $atTime: AtTime, $target: String) {
	project(organization: $organization, project: $project) {
		... on Project {
			evaluate(expr: $expr, atTime: $atTime, target: $target) {
				... on Build {
					__typename
					status
				}
				... on Error {
					__typename
					message
				}
				... on ErrorWithSubErrors {
					__typename
					subErrors {
						__typename
						... on GenericSolveError {
							message
							isTransient
							validationErrors {
								error
								jsonPath
							}
						}
						... on RemediableSolveError {
							message
							isTransient
							errorType
							validationErrors {
								error
								jsonPath
							}
						}
					}
				}
			}
		}
	}
}
`
}

func (b *evaluate) Vars() (map[string]interface{}, error) {
	return b.vars, nil
}
