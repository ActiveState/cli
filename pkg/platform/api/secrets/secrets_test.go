package secrets_test

import (
	"fmt"
	"testing"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal-as/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal-as/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
)

func TestSecretsAPI_NewClient_Success(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	serviceURL := api.GetServiceURL(api.ServiceSecrets)
	client := secretsapi.NewDefaultClient()
	require.NotNil(client)
	assert.Equal(fmt.Sprintf("%s://%s%s", serviceURL.Scheme, serviceURL.Host, serviceURL.Path), client.BaseURI)

	rt, isRuntime := client.Transport.(*httptransport.Runtime)
	require.True(isRuntime, "client.Transport is a Runtime")
	assert.Equal(serviceURL.Host, rt.Host)
	assert.Equal(serviceURL.Path, rt.BasePath)

	// validate that the client.Auth writer sets the bearer token using the one we provided
	mockClientRequest := new(MockClientRequest)
	mockClientRequest.On("SetHeaderParam", "Authorization", []string{"Bearer bearer123"}).Return(nil)
}

func TestSecretsAPI_InitializeClient_Success(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	serviceURL := api.GetServiceURL(api.ServiceSecrets)
	secretsapi.InitializeClient()

	client := secretsapi.Get()
	require.NotNil(client)
	assert.Equal(fmt.Sprintf("%s://%s%s", serviceURL.Scheme, serviceURL.Host, serviceURL.Path), client.BaseURI)

	rt, isRuntime := client.Transport.(*httptransport.Runtime)
	require.True(isRuntime, "client.Transport is a Runtime")
	assert.Equal(serviceURL.Host, rt.Host)
	assert.Equal(serviceURL.Path, rt.BasePath)

	// validate that the client.Auth writer sets the bearer token using the one we provided
	mockClientRequest := new(MockClientRequest)
	mockClientRequest.On("SetHeaderParam", "Authorization", []string{"Bearer "}).Return(nil)
}

func TestSecretsAPI_Authenticated_Failure(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	client := secretsapi_test.NewDefaultTestClient("bearer123")
	require.NotNil(client)

	httpmock.Activate(client.BaseURI)
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("GET", "/whoami", 401)

	uid, err := client.AuthenticatedUserID()
	assert.Zero(uid)
	require.Error(err)
}

func TestSecretsAPI_Authenticated_Success(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	client := secretsapi_test.NewDefaultTestClient("bearer123")
	require.NotNil(client)

	httpmock.Activate(client.BaseURI)
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("GET", "/whoami", 200)

	uid, err := client.AuthenticatedUserID()
	assert.Nil(err)
	assert.Equal("11110000-1111-0000-1111-000011110000", uid.String())
}
