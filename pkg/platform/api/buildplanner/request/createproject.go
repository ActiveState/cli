package request

import "github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"

func CreateProject(owner, project string, private bool, expr *buildexpression.BuildExpression, description string) *createProject {
	return &createProject{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"private":      private,
		"expr":         expr,
		"description":  description,
	}}
}

type createProject struct {
	vars map[string]interface{}
}

func (c *createProject) Query() string {
	return `
mutation ($organization: String!, $project: String!, $private: Boolean!, $expr: BuildExpr!, $description: String!) {
	createProject(input:{organization:$organization, project:$project, private:$private, expr:$expr, description:$description}) {
		... on ProjectCreated {
			__typename
			commit {
				__typename
				commitId
			}
		}
		... on AlreadyExists {
			__typename
			message
		}
		... on NotFound {
			__typename
			message
		}
		... on ParseError {
			__typename
			message
			path
		}
		... on ValidationError {
			__typename
			message
		}
		... on Forbidden {
			__typename
			message
		}
	}
}`
}

func (c *createProject) Vars() map[string]interface{} {
	return c.vars
}