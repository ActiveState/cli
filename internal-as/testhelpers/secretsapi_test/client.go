package secretsapi_test

import (
	"net/http"

	httptransport "github.com/go-openapi/runtime/client"

	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
)

var testTransport http.RoundTripper

// NewTestClient creates a new secretsapi.Client with a testable Transport. Makes it possible to
// use httpmock.
func NewTestClient(scheme, host, basePath string) *secretsapi.Client {
	return withTestableTransport(secretsapi.NewClient(scheme, host, basePath))
}

// NewDefaultTestClient creates a testable secrets client using environment defaults for schema, host, and path.
func NewDefaultTestClient(bearerToken string) *secretsapi.Client {
	return withTestableTransport(secretsapi.NewDefaultClient())
}

// InitializeTestClient initializes a testable secrets client using environment defaults for schema,
// host, and path. While this function departs from the secretsapi.InitializeClient signature, it's
// more useful for testing.
func InitializeTestClient(bearerToken string) *secretsapi.Client {
	secretsapi.DefaultClient = NewDefaultTestClient(bearerToken)
	return secretsapi.DefaultClient
}

func withTestableTransport(client *secretsapi.Client) *secretsapi.Client {
	rt := client.Transport.(*httptransport.Runtime)
	rt.Transport = testTransport
	return client
}
