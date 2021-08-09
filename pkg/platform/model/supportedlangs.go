package model

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/mediator"
	"github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	"github.com/ActiveState/cli/pkg/platform/api/mediator/request"
)

// FetchSupportedLanguages returns the list of languages that the Platform supports ATM
func FetchSupportedLanguages() ([]model.SupportedLanguage, error) {
	req := request.SupportedLanguages()
	var resp model.SupportedLanguagesResponse
	med := mediator.New(nil)
	err := med.Run(req, &resp)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to run mediator request.")
	}
	return resp.Languages, nil
}
