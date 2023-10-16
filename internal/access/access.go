package access

import (
	"errors"

	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// Secrets determines whether the authorized user has access
// to the current project's secrets
func Secrets(orgName string, auth *authentication.Auth) (bool, error) {
	_, err := model.FetchOrgMember(orgName, auth.WhoAmI(), auth)
	if err != nil {
		if errors.Is(err, model.ErrMemberNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
