package export

import (
	"errors"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
)

// APIKeyProvider describes the behavior required to obtain a new api key.
type APIKeyProvider interface {
	NewAPIKey(string) (string, *failures.Failure)
}

// APIKeyRunParams manages the request-specific parameters used to run the
// primary APIKey logic.
type APIKeyRunParams struct {
	Name     string
	IsAuthed func() bool
}

func prepareAPIKeyRunParams(params APIKeyRunParams) (APIKeyRunParams, error) {
	if params.Name == "" {
		return params, errors.New(locale.T("err_apikey_name_required"))
	}

	if !params.IsAuthed() {
		return params, errors.New(locale.T("err_command_requires_auth"))
	}

	return params, nil
}

// APIKey manages the core dependencies for the primary APIKey logic.
type APIKey struct {
	keyPro APIKeyProvider
	out    output.Outputer
}

// NewAPIKey is a convenience construction function.
func NewAPIKey(keyPro APIKeyProvider, out output.Outputer) *APIKey {
	return &APIKey{
		out:    out,
		keyPro: keyPro,
	}
}

// Run executes the primary APIKey logic.
func (k *APIKey) Run(params APIKeyRunParams) error {
	logging.Debug("Execute export API key")

	ps, err := prepareAPIKeyRunParams(params)
	if err != nil {
		return failures.FailUser.New(err.Error())
	}

	key, fail := k.keyPro.NewAPIKey(ps.Name)
	if err != nil {
		return fail.WithDescription("err_cannot_obtain_apikey")
	}

	k.out.Notice(locale.T("export_apikey_user_notice"))
	k.out.Print(key)
	return nil
}
