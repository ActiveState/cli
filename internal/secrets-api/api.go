package secretsapi

import (
	"fmt"

	"github.com/ActiveState/cli/internal/api"
	apiEnv "github.com/ActiveState/cli/internal/api/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/secrets-api/client"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

var (
	// FailNotFound indicates a failure to find a user's resource.
	FailNotFound = failures.Type("secrets-api.fail.not_found", failures.FailUser)

	// FailKeypairNotFound indicates a failure to find a keypair.
	FailKeypairNotFound = failures.Type("secrets-api.fail.keypair.not_found", FailNotFound)

	// FailPublicKeyNotFound indicates a failure to find a public-key.
	FailPublicKeyNotFound = failures.Type("secrets-api.fail.publickey.not_found", FailNotFound)

	// FailUserSecretNotFound indicates a failure to find a user secret.
	FailUserSecretNotFound = failures.Type("secrets-api.fail.user_secret.not_found", FailNotFound)

	// FailSave indicates a failure to save a user's resource.
	FailSave = failures.Type("secrets-api.fail.save", failures.FailUser)

	// FailKeypairSave indicates a failure to save a keypair.
	FailKeypairSave = failures.Type("secrets-api.fail.keypair.save", FailSave)

	// FailUserSecretSave indicates a failure to save a user secret.
	FailUserSecretSave = failures.Type("secrets-api.fail.user_secret.save", FailSave)
)

// Client encapsulates a Secrets Service API client and its configuration
type Client struct {
	*client.Secrets
	BaseURI string
	Auth    runtime.ClientAuthInfoWriter
}

// NewClient creates a new SecretsAPI client instance using the provided HTTP settings.
// It also expects to receive the actual Bearer token value that will be passed in each
// API request in order to authenticate each request.
func NewClient(schema, host, basePath, bearerToken string) *Client {
	logging.Debug("secrets-api scheme=%s host=%s base_path=%s", schema, host, basePath)
	secretsClient := &Client{
		Secrets: client.New(httptransport.New(host, basePath, []string{schema}), strfmt.Default),
		BaseURI: fmt.Sprintf("%s://%s%s", schema, host, basePath),
		Auth:    httptransport.BearerToken(bearerToken),
	}
	return secretsClient
}

// NewDefaultClient creates a new Client using constants SecretsAPISchema, -Host, and -Path.
func NewDefaultClient(bearerToken string) *Client {
	apiSetting := apiEnv.GetSecretsAPISettings()
	return NewClient(apiSetting.Schema, apiSetting.Host, apiSetting.BasePath, bearerToken)
}

// Authenticated will check with the Secrets Service to ensure the current Bearer token is a valid
// one and return the user's UID in the response. Otherwise, this function will return a Failure.
func (client *Client) Authenticated() (*strfmt.UUID, *failures.Failure) {
	resOk, err := client.Authentication.GetWhoami(nil, client.Auth)
	if err != nil {
		if api.ErrorCode(err) == 401 {
			return nil, api.FailAuth.New("err_api_not_authenticated")
		}
		return nil, api.FailAuth.Wrap(err)
	}
	return resOk.Payload.UID, nil
}
