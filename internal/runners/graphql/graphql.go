package graphql

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/graphql"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type Runner struct {
	out  output.Outputer
	auth *authentication.Auth
	api  api.Service
}

type Request struct {
	QueryStr  string
	QueryVars map[string]interface{}
}

// Query() and Vars() implement /cli/internal/gqlclient/gqlclient.Request interface
func (req *Request) Query() string {
	return req.QueryStr
}

func (req *Request) Vars() (map[string]interface{}, error) {
	return req.QueryVars, nil
}

func New(prime *primer.Values, api api.Service) *Runner {
	return &Runner{
		out:  prime.Output(),
		auth: prime.Auth(),
		api:  api,
	}
}

func (runner *Runner) Run(request *Request, response interface{}) error {
	gql := graphql.New(runner.auth, runner.api)

	err := gql.Run(request, &response)
	if err != nil {
		return errs.Wrap(err, "gql.Run failed")
	}

	return nil
}
