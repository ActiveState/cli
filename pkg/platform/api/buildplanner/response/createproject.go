package response

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

type ProjectCreated struct {
	Type   string  `json:"__typename"`
	Commit *Commit `json:"commit"`
	*Error
}

type ProjectCreatedError struct {
	Type    string
	Message string
}

func (p *ProjectCreatedError) Error() string { return p.Message }

func ProcessProjectCreatedError(pcErr *ProjectCreated, fallbackMessage string) error {
	if pcErr.Error == nil {
		return errs.New(fallbackMessage)
	}

	if pcErr.Type == types.InvalidInputErrorType {
		return locale.NewInputError("err_buildplanner_create_project", "Could not create project. Received message: {{.V0}}", pcErr.Message)
	}

	return &ProjectCreatedError{pcErr.Type, pcErr.Message}
}
