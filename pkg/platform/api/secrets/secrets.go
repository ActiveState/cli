package secrets

import (
	"fmt"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_client"
	secretsapiClient "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_client/secrets"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
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

// Scope covers what scope a secret belongs to
type Scope string

var (
	// ScopeUser is the user scope
	ScopeUser Scope = "user"

	//ScopeProject is the project scope
	ScopeProject Scope = "project"
)

var persistentClient *Client

// Client encapsulates a Secrets Service API client and its configuration
type Client struct {
	*secrets_client.Secrets
	BaseURI string
}

// GetClient gets the cached (if any) client instance that was initialized using our default settings
func GetClient() *Client {
	if persistentClient == nil {
		persistentClient = NewDefaultClient()
	}
	return persistentClient
}

// Reset will reset the client cache
func Reset() {
	persistentClient = nil
}

// NewClient creates a new SecretsAPI client instance using the provided HTTP settings.
// It also expects to receive the actual Bearer token value that will be passed in each
// API request in order to authenticate each request.
func NewClient(schema, host, basePath string) *Client {
	logging.Debug("secrets-api scheme=%s host=%s base_path=%s", schema, host, basePath)
	transportRuntime := httptransport.New(host, basePath, []string{schema})
	//transportRuntime.SetDebug(true)
	secretsClient := &Client{
		Secrets: secrets_client.New(transportRuntime, strfmt.Default),
		BaseURI: fmt.Sprintf("%s://%s%s", schema, host, basePath),
	}
	return secretsClient
}

// NewDefaultClient creates a new Client using constants SecretsAPISchema, -Host, and -Path and
// a provided Bearer-token value.
func NewDefaultClient() *Client {
	serviceURL := api.GetServiceURL(api.ServiceSecrets)
	return NewClient(serviceURL.Scheme, serviceURL.Host, serviceURL.Path)
}

// DefaultClient represents a secretsapi Client instance that can be accessed by any package
// needing it. DefaultClient should be set by a call to InitializeClient; this, it can be nil.
var DefaultClient *Client

// InitializeClient will create new Client using defaults, including the api.BearerToken value.
// This new Client instance will be accessible as secretapi.DefaultClient afterwards. Calling
// this function multiple times will redefine the DefaultClient value using the defaults/constants
// available to it at the time of the call; thus, the DefaultClient can be re-initialized this way.
// Because this function is dependent on a runtime-value from pkg/platform/api, we are not relying on
// the init() function for instantiation; this must be called explicitly.
func InitializeClient() *Client {
	DefaultClient = NewDefaultClient()
	return DefaultClient
}

// Get is an alias for InitializeClient used to persist our Get() pattern used throughout the codebase
func Get() *Client {
	return InitializeClient()
}

// AuthenticatedUserID will check with the Secrets Service to ensure the current Bearer token
// is a valid one and return the user's UID in the response. Otherwise, this function will return
// a Failure.
func (client *Client) AuthenticatedUserID() (strfmt.UUID, error) {
	resOk, err := client.Authentication.GetWhoami(nil, authentication.Get().ClientAuth())
	if err != nil {
		if api.ErrorCode(err) == 401 {
			return "", api.FailAuth.New("err_api_not_authenticated")
		}
		return "", api.FailAuth.Wrap(err)
	}
	return *resOk.Payload.UID, nil
}

// Persist will make the current client the persistentClient
func (client *Client) Persist() {
	persistentClient = client
}

// FetchAll fetchs the current user's secrets for an organization.
func FetchAll(client *Client, org *mono_models.Organization) ([]*secretsModels.UserSecret, error) {
	params := secretsapiClient.NewGetAllUserSecretsParams()
	params.OrganizationID = org.OrganizationID
	getOk, err := client.Secrets.Secrets.GetAllUserSecrets(params, authentication.Get().ClientAuth())
	if err != nil {
		switch statusCode := api.ErrorCode(err); statusCode {
		case 401:
			return nil, api.FailAuth.New("err_api_not_authenticated")
		default:
			return nil, api.FailUnknown.Wrap(err)
		}
	}
	return getOk.Payload, nil
}

// FetchDefinitions fetchs the secret definitions for a given project.
func FetchDefinitions(client *Client, projectID strfmt.UUID) ([]*secretsModels.SecretDefinition, error) {
	params := secretsapiClient.NewGetDefinitionsParams()
	params.ProjectID = projectID
	getOk, err := client.Secrets.Secrets.GetDefinitions(params, authentication.Get().ClientAuth())
	if err != nil {
		switch statusCode := api.ErrorCode(err); statusCode {
		case 401:
			return nil, api.FailAuth.New("err_api_not_authenticated")
		default:
			return nil, api.FailUnknown.Wrap(err)
		}
	}
	return getOk.Payload, nil
}

func SaveSecretShares(client *Client, org *mono_models.Organization, user *mono_models.User, shares []*secretsModels.UserSecretShare) error {
	params := secretsapiClient.NewShareUserSecretsParams()
	params.OrganizationID = org.OrganizationID
	params.UserID = user.UserID
	params.UserSecrets = shares
	_, err := client.Secrets.Secrets.ShareUserSecrets(params, authentication.Get().ClientAuth())
	if err != nil {
		logging.Debug("error sharing user secrets with %s: %v", user.Username, err)
		return FailSave.New("secrets_err_save")
	}
	return nil
}
