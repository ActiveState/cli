package auth

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// promptConfirm will prompt for a yes/no confirmation and return true if confirmed.
func promptConfirm(translationID string) (bool, *failures.Failure) {
	resp, fail := prompt.Confirm(locale.T(translationID))
	if fail != nil {
		return false, fail
	}
	return resp, nil
}

// Logout will clear any stored credentials
func Logout() {
	authentication.Logout()
	keypairs.DeleteWithDefaults()
}
