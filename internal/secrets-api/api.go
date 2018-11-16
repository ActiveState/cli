package secretsapi

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/secrets-api/client"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

var (
	// ErrNoBearerToken is an error that reflects no bearer token provided when
	// configuring Client
	ErrNoBearerToken = errors.New("no bearer token provided")

	// FailNotFound indicates a failure to find a user's resource.
	FailNotFound = failures.Type("secrets-api.fail.not_found", failures.FailUser)
	// FailSave indicates a failure to save a user's resource.
	FailSave = failures.Type("secrets-api.fail.save", failures.FailUser)
)

// Client ...
type Client struct {
	*client.Secrets
	BaseURI string
	Auth    runtime.ClientAuthInfoWriter
}

// NewClient creates a new SecretsAPI client instance using the provided HTTP settings.
// It also expects to receive the actual Bearer token value that will be passed in each
// API request in order to authenticate each request.
func NewClient(scheme, host, basePath, bearerToken string) *Client {
	secretsClient := &Client{
		Secrets: client.New(httptransport.New(host, basePath, []string{scheme}), strfmt.Default),
		BaseURI: fmt.Sprintf("%s://%s%s", scheme, host, basePath),
		Auth:    httptransport.BearerToken(bearerToken),
	}
	return secretsClient
}

var testTransport http.RoundTripper

// NewTestClient ...
// TODO move this into a test helper
func NewTestClient(scheme, host, basePath, bearerToken string) *Client {
	newClient := NewClient(scheme, host, basePath, bearerToken)
	// this is necessary to allow httpmock tests to function
	rt := newClient.Transport.(*httptransport.Runtime)
	rt.Transport = testTransport
	return newClient
}

// Authenticated will check with the Secrets Service to ensure the current Bearer token is a valid
// one and return the user's UID in the response. Otherwise, this function will return a Failure.
func (client *Client) Authenticated() (*strfmt.UUID, *failures.Failure) {
	resOk, err := client.Authentication.GetWhoami(nil, client.Auth)
	if err != nil {
		if ErrorCode(err) == 401 {
			return nil, api.FailAuth.New("err_api_not_authenticated")
		}
		return nil, api.FailUnknown.Wrap(err)
	}
	return resOk.Payload.UID, nil
}

// ErrorCode tries to retrieve the code associated with an API error. ErrorCode assumes
// the actual err object is a models.Message. At some point, this should be changed to not
// use reflection, but it is modeled off of api.ErrorCode. The difference here is that the
// Secrets Service API always defines non-2XX responses with a models.Message type.
func ErrorCode(err interface{}) int {
	r := reflect.ValueOf(err)
	payload := reflect.Indirect(r).FieldByName("Payload")
	if !payload.IsValid() {
		return -1
	}

	codeptr := reflect.Indirect(payload).FieldByName("Code")
	if !codeptr.IsValid() {
		return -1
	}

	code := reflect.Indirect(codeptr)
	if !code.IsValid() {
		return -1
	}
	return int(code.Int())
}
