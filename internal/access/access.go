package access

import (
	"errors"

	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// Secrets determines whether the authorized user has access
// to the current project's secrets
func Secrets(orgName string) (bool, error) {
	if isProjectOwner(orgName) {
		return true, nil
	}

	return isOrgMember(orgName)
}

func isProjectOwner(orgName string) bool {
	auth := authentication.LegacyGet()
	if orgName != auth.WhoAmI() {
		return false
	}
	return true
}

func isOrgMember(orgName string) (bool, error) {
	auth := authentication.LegacyGet()
	_, err := model.FetchOrgMember(orgName, auth.WhoAmI())
	if err != nil {
		if errors.Is(err, model.ErrMemberNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
