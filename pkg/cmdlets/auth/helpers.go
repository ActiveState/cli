package auth

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

// promptConfirm will prompt for a yes/no confirmation and return true if confirmed.
func promptConfirm(translationID string) (confirmed bool) {
	survey.AskOne(&survey.Confirm{
		Message: locale.T(translationID),
	}, &confirmed, nil)
	return confirmed
}

// Logout will clear any stored credentials
func Logout() {
	authentication.Logout()
	keypairs.DeleteWithDefaults()
}
