package legalprompt

import (
	"io"
	"net/http"
	"strings"

	"github.com/ActiveState/cli/internal-as/constants"
	"github.com/ActiveState/cli/internal-as/errs"
	"github.com/ActiveState/cli/internal-as/locale"
	"github.com/ActiveState/cli/internal-as/output"
	"github.com/ActiveState/cli/internal-as/prompt"
	"github.com/ActiveState/cli/internal-as/rtutils/p"
)

func DownloadTOS() (string, error) {
	resp, err := http.Get(constants.TermsOfServiceURLText)
	if err != nil {
		return "", errs.Wrap(err, "Failed to download the Terms Of Service document.")
	}
	if resp.StatusCode != http.StatusOK {
		return "", errs.New("The server responded with status '%s' when trying to download the Terms Of Service document.", resp.Status)
	}
	defer resp.Body.Close()

	tosText := new(strings.Builder)
	_, err = io.Copy(tosText, resp.Body)
	if err != nil {
		return "", errs.Wrap(err, "Failed to read Terms Of Service contents.")
	}

	return tosText.String(), nil
}

func TOS(out output.Outputer, prompt prompt.Prompter) (bool, error) {
	choices := []string{
		locale.T("tos_accept"),
		locale.T("tos_not_accept"),
		locale.T("tos_show_full"),
	}

	out.Notice(locale.Tr("tos_disclaimer", constants.TermsOfServiceURLLatest))
	defaultChoice := locale.T("tos_accept")
	choice, err := prompt.Select(locale.Tl("tos", "Terms of Service"), locale.T("tos_acceptance"), choices, &defaultChoice)
	if err != nil {
		return false, err
	}

	switch choice {
	case locale.T("tos_accept"):
		return true, nil
	case locale.T("tos_not_accept"):
		return false, nil
	case locale.T("tos_show_full"):
		tosText, err := DownloadTOS()
		if err != nil {
			return false, locale.WrapInputError(err, "err_download_tos")
		}
		out.Print(tosText)
		return prompt.Confirm("", locale.T("tos_acceptance"), p.BoolP(true))
	}

	return false, nil
}
