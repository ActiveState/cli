package access

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// Secrets determines whether the authorized user has access
// to the current project's secrets
func Secrets(orgName string) (bool, *failures.Failure) {
	if isProjectOwner(orgName) {
		return true, nil
	}

	return isOrgMember(orgName)
}

func isProjectOwner(orgName string) bool {
	auth := authentication.Get()
	if orgName != auth.WhoAmI() {
		return false
	}
	return true
}

func isOrgMember(orgName string) (bool, *failures.Failure) {
	auth := authentication.Get()
	_, fail := model.FetchOrgMember(orgName, auth.WhoAmI())
	if fail != nil {
		if api.FailNotFound.Matches(fail.Type) {
			return false, nil
		}
		return false, fail
	}

	return true, nil
}
