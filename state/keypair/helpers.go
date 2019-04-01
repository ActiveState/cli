package keypair

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/prompt"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

func promptForPassphrase() (string, *failures.Failure) {
	var passphrase string
	var prompter = &survey.Password{Message: locale.T("passphrase_prompt")}
	if err := survey.AskOne(prompter, &passphrase, prompt.ValidateRequired); err != nil {
		return "", FailInputPassphrase.New("keypair_err_passphrase_prompt")
	}
	return passphrase, nil
}
