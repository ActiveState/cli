package secretsapi

import (
	"fmt"

	"github.com/ActiveState/cli/pkg/platform/authentication"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/secrets-api/client"
	"github.com/ActiveState/cli/pkg/platform/api"
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

// NewDefaultClient creates a new Client using constants SecretsAPISchema, -Host, and -Path and
// a provided Bearer-token value.
func NewDefaultClient(bearerToken string) *Client {
	apiSetting := api.GetSettings(api.ServiceSecrets)
	return NewClient(apiSetting.Schema, apiSetting.Host, apiSetting.BasePath, bearerToken)
}

// DefaultClient represents a secretsapi Client instance that can be accessed by any package
// needing it. DefaultClient should be set by a call to InitializeClient; this, it can be nil.
var DefaultClient *Client

// InitializeClient will create new Client using defaults, including the api.BearerToken value.
// This new Client instance will be accessible as secretapi.DefaultClient afterwards. Calling
// this function multiple times will redefine the DefaultClient value using the defaults/constants
// available to it at the time of the call; thus, the DefaultClient can be re-initialized this way.
// Because this function is dependent on a runtime-value from internal/api, we are not relying on
// the init() function for instantiation; this must be called explicitly.
func InitializeClient() *Client {
	DefaultClient = NewDefaultClient(authentication.Get().BearerToken())
	return DefaultClient
}

// AuthenticatedUserID will check with the Secrets Service to ensure the current Bearer token
// is a valid one and return the user's UID in the response. Otherwise, this function will return
// a Failure.
func (client *Client) AuthenticatedUserID() (strfmt.UUID, *failures.Failure) {
	resOk, err := client.Authentication.GetWhoami(nil, client.Auth)
	if err != nil {
		if api.ErrorCode(err) == 401 {
			return "", api.FailAuth.New("err_api_not_authenticated")
		}
		return "", api.FailAuth.Wrap(err)
	}
	return *resOk.Payload.UID, nil
}
