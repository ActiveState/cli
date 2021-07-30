package model

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/mediator"
	"github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	"github.com/ActiveState/cli/pkg/platform/api/mediator/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// FetchSupportedLanguages returns the list of languages that the Platform supports ATM
func FetchSupportedLanguages(auth *authentication.Auth) ([]string, error) {
	req := request.SupportedLanguages()
	var resp model.SupportedLanguagesResponse
	med := mediator.New(auth)
	err := med.Run(req, &resp)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to run mediator request.")
	}
	return resp.Languages, nil
}
