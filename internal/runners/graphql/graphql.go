package graphql

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type Runner struct {
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

func New(auth *authentication.Auth, api api.Service) *Runner {
	return &Runner{
		auth: auth,
		api:  api,
	}
}

func (runner *Runner) Run(request *Request, response interface{}) error {
	gqlUrl := api.GetServiceURL(runner.api)
	gql := gqlclient.NewWithOpts(gqlUrl.String(), 0, gqlclient.WithHTTPClient(api.NewHTTPClient()))
	gql.SetTokenProvider(runner.auth)

	err := gql.Run(request, &response)
	if err != nil {
		return errs.Wrap(err, "gql.Run failed: "+err.Error())
	}

	return nil
}
