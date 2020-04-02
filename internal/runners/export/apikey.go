package export

import (
	"errors"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

// APIKeyProvider describes the behavior required to obtain a new api key.
type APIKeyProvider interface {
	NewAPIKey(string) (string, *failures.Failure)
}

// Printer describes a basic print provider.
type Printer interface {
	Print(interface{})
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
	out    Printer
}

// NewAPIKey is a convenience construction function.
func NewAPIKey(keyPro APIKeyProvider, out Printer) *APIKey {
	return &APIKey{
		out:    out,
		keyPro: keyPro,
	}
}

// Run executes the primary APIKey logic.
func (k *APIKey) Run(params APIKeyRunParams) error {
	return runAPIKey(k.keyPro, k.out, params)
}

func runAPIKey(keyPro APIKeyProvider, out Printer, params APIKeyRunParams) error {
	logging.Debug("Execute export API key")

	ps, err := prepareAPIKeyRunParams(params)
	if err != nil {
		return failures.FailUser.New(err.Error())
	}

	key, fail := keyPro.NewAPIKey(ps.Name)
	if err != nil {
		return fail.WithDescription("err_cannot_obtain_apikey")
	}

	out.Print(key)
	return nil
}
