package access

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// Secrets determines whether the authorized user has access
// to the current project's secrets
func Secrets() (bool, *failures.Failure) {
	if isProjectOwner() {
		return true, nil
	}

	return isOrgMember()
}

func isProjectOwner() bool {
	project := project.Get()
	auth := authentication.Get()
	if project.Owner() != auth.WhoAmI() {
		return false
	}
	return true
}

func isOrgMember() (bool, *failures.Failure) {
	project := project.Get()
	org, fail := model.FetchOrgByURLName(project.Owner())
	if fail != nil {
		return false, fail
	}

	auth := authentication.Get()
	_, fail = model.FetchOrgMember(org, auth.WhoAmI())
	if fail != nil {
		if api.FailNotFound.Matches(fail.Type) {
			return false, nil
		}
		return false, fail
	}

	return true, nil
}
