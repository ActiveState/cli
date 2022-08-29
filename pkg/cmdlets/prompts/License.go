package prompts

import (
    "os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/output"
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
		return "", errs.Wrap(err, "Unable to open TOS file")
	}

	return string(b), nil
}

func PromptOfflineLicense(out output.Outputer, prompt prompt.Prompter, filepath string)(bool,error) {
    legalText := newOfflineLicense(filepath)
    return PromptTOS(legalText,out,prompt)
}
