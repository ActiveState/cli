package response

import "github.com/ActiveState/cli/internal/errs"

// ProjectResponse contains the commit and any errors.
type ProjectResponse struct {
	Type   string  `json:"__typename"`
	Commit *Commit `json:"commit"`
	*Error
}

// PostProcess must satisfy gqlclient.PostProcessor interface
func (pr *ProjectResponse) PostProcess() error {
	if pr == nil {
		return errs.New("Project is nil")
	}

	if IsErrorResponse(pr.Type) {
		return ProcessProjectError(pr, "Could not get build from project response")
	}

	if pr.Commit == nil {
		return errs.New("Commit is nil")
	}

	if IsErrorResponse(pr.Type) {
		return ProcessProjectError(pr, "Could not get build from project response")
	}

	if pr.Commit == nil {
		return errs.New("Commit is nil")
	}

	if IsErrorResponse(pr.Commit.Type) {
		return ProcessCommitError(pr.Commit, "Could not get build from commit from project response")
	}

	if pr.Commit.Build == nil {
		return errs.New("Commit does not contain build")
	}

	if IsErrorResponse(pr.Commit.Build.Type) {
		return ProcessBuildError(pr.Commit.Build, "Could not get build from project commit response")
	}

	return nil
}
