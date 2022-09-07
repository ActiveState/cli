package legalprompt

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
)

func CustomLicense(text string, out output.Outputer, prompt prompt.Prompter) (bool, error) {
	choices := []string{
		locale.T("lic_accept"),
		locale.T("lic_not_accept"),
		locale.T("lic_show_full"),
	}

	defaultChoice := locale.T("lic_accept")
	choice, err := prompt.Select(locale.Tl("license", "Accept License"), locale.T("lic_acceptance"), choices, &defaultChoice)
	if err != nil {
		return false, err
	}

	switch choice {
	case locale.T("lic_accept"):
		return true, nil
	case locale.T("lic_not_accept"):
		return false, nil
	case locale.T("lic_show_full"):
		out.Print(text)

		tosConfirmDefault := true
		return prompt.Confirm("", locale.T("lic_acceptance"), &tosConfirmDefault)
	}

	return false, nil
}
