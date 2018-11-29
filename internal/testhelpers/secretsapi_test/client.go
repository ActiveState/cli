package secretsapi_test

import (
	"net/http"

	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	httptransport "github.com/go-openapi/runtime/client"
)

var testTransport http.RoundTripper

// NewTestClient creates a new secretsapi.Client with a testable Transport. Makes it possible to
// use httpmock.
func NewTestClient(scheme, host, basePath, bearerToken string) *secretsapi.Client {
	return withTestableTransport(secretsapi.NewClient(scheme, host, basePath, bearerToken))
}

// NewDefaultTestClient creates a testable secrets client using environment defaults for schema, host, and path.
func NewDefaultTestClient(bearerToken string) *secretsapi.Client {
	return withTestableTransport(secretsapi.NewDefaultClient(bearerToken))
}

func withTestableTransport(client *secretsapi.Client) *secretsapi.Client {
	rt := client.Transport.(*httptransport.Runtime)
	rt.Transport = testTransport
	return client
}
