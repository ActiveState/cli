package keypair

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/prompt"
)

var prompter prompt.Prompter

func init() {
	prompter = prompt.New()
}

func promptForPassphrase() (string, *failures.Failure) {
	var passphrase string
	passphrase, fail := prompter.InputPassword(locale.T("passphrase_prompt"))
	if fail != nil {
		return "", FailInputPassphrase.New("keypair_err_passphrase_prompt")
	}
	return passphrase, nil
}
