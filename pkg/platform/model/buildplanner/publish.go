package buildplanner

import (
	"github.com/ActiveState/cli/internal/errs"
	graphModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
)

func (b *BuildPlanner) Publish(vars request.PublishVariables, filepath string) (*graphModel.PublishResult, error) {
	pr, err := request.Publish(vars, filepath)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create publish request")
	}
	res := graphModel.PublishResult{}

	if err := b.client.Run(pr, &res); err != nil {
		return nil, processBuildPlannerError(err, "Publish failed")
	}

	if res.Error != "" {
		return nil, errs.New("API responded with error: %s", res.Error)
	}

	return &res, nil
}
