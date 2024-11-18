package response

import (
	"github.com/ActiveState/cli/internal/errs"
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

	return &ProjectCreatedError{pcErr.Type, pcErr.Message}
}
