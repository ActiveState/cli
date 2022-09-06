package prompts

import (
    "os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/prompt"
)


type offlineLicense struct {
	LegalText
    filepath string
}

func newOfflineLicense(filepath string) *offlineLicense {
    return &offlineLicense{filepath: filepath}
}

func (license *offlineLicense) GetLegalText() (string, error) {
    b, err := os.ReadFile(license.filepath)
	if err != nil {
		return "", errs.Wrap(err, "Unable to open License file")
	}

	return string(b), nil
}

func PromptOfflineLicense(out output.Outputer, prompt prompt.Prompter, filepath string)(bool,error) {
    legalText := newOfflineLicense(filepath)
    return PromptLicense(legalText,out,prompt)
}

func PromptLicense(lic LegalText, out output.Outputer, prompt prompt.Prompter) (bool, error) {
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
		tosText, err := lic.GetLegalText()
		if err != nil {
			return false, locale.WrapError(err, "err_get_license_text", "Could not get license text.")
		}

		out.Print(tosText)

		tosConfirmDefault := true
		return prompt.Confirm("", locale.T("lic_acceptance"), &tosConfirmDefault)
	}

	return false, nil
}
