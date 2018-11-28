package secretsapi_test

import (
	"net/http"

	"github.com/ActiveState/cli/internal/constants"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	httptransport "github.com/go-openapi/runtime/client"
)

var testTransport http.RoundTripper

// NewTestClient creates a new secretsapi.Client with a testable Transport. Makes it possible to
// use httpmock.
func NewTestClient(scheme, host, basePath, bearerToken string) *secretsapi.Client {
	newClient := secretsapi.NewClient(scheme, host, basePath, bearerToken)
	// this is necessary to allow httpmock tests to function
	rt := newClient.Transport.(*httptransport.Runtime)
	rt.Transport = testTransport
	return newClient
}

// NewDefaultTestClient creates a testable secrets client using constants for schema, host, and path.
func NewDefaultTestClient(bearerToken string) *secretsapi.Client {
	return NewTestClient(constants.SecretsAPISchema, constants.SecretsAPIHost, constants.SecretsAPIPath, bearerToken)
}
