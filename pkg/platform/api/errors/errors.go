package api_errors

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/rollbar"
)

type ErrCountryBlocked struct{ *locale.LocalizedError }

func NewCountryBlockedError() *ErrCountryBlocked {
	rollbar.DoNotReportMessages.Add(locale.T("err_country_blocked"))
	return &ErrCountryBlocked{LocalizedError: locale.NewExternalError("err_country_blocked")}
}
