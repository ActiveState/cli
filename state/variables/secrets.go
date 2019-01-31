package variables

import (
	"github.com/ActiveState/cli/internal/locale"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
)

func secretScopeDescription(userSecret *secretsModels.UserSecret) string {
	if *userSecret.IsUser && userSecret.ProjectID != "" {
		return locale.T("variables_scope_user_project")
	} else if *userSecret.IsUser {
		return locale.T("variables_scope_user_org")
	} else if userSecret.ProjectID != "" {
		return locale.T("variables_scope_project")
	} else {
		return locale.T("variables_scope_org")
	}
}
