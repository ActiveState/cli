package response

import (
	"github.com/ActiveState/cli/internal/errs"
)

type projectCreated struct {
	Type   string  `json:"__typename"`
	Commit *Commit `json:"commit"`
	*Error
	*NotFoundError
	*ParseError
	*ForbiddenError
}

type CreateProjectResult struct {
	ProjectCreated *projectCreated `json:"createProject"`
}

type ProjectCreatedError struct {
	Type    string
	Message string
}

func (p *ProjectCreatedError) Error() string { return p.Message }

func ProcessProjectCreatedError(pcErr *projectCreated, fallbackMessage string) error {
	if pcErr.Error == nil {
		return errs.New(fallbackMessage)
	}

	return &ProjectCreatedError{pcErr.Type, pcErr.Message}
}
