package auth

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func SignupWithBrowser(out output.Outputer, auth *authentication.Auth, prompt prompt.Prompter) error {
	logging.Debug("Signing up with browser")

	err := authenticateWithBrowser(out, auth, prompt, true)
	if err != nil {
		return errs.Wrap(err, "Error signing up with browser")
	}

	out.Notice(locale.Tl("auth_signup_success", "Successfully signed up and authorized this device"))

	return nil
}
