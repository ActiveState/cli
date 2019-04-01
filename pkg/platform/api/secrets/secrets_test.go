package secrets_test

import (
	"fmt"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretsAPI_NewClient_Success(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	apiSetting := api.GetSettings(api.ServiceSecrets)
	client := secretsapi.NewDefaultClient("bearer123")
	require.NotNil(client)
	assert.NotNil(client.Auth)
	assert.Equal(fmt.Sprintf("%s://%s%s", apiSetting.Schema, apiSetting.Host, apiSetting.BasePath), client.BaseURI)

	rt, isRuntime := client.Transport.(*httptransport.Runtime)
	require.True(isRuntime, "client.Transport is a Runtime")
	assert.Equal(apiSetting.Host, rt.Host)
	assert.Equal(apiSetting.BasePath, rt.BasePath)

	// validate that the client.Auth writer sets the bearer token using the one we provided
	mockClientRequest := new(MockClientRequest)
	mockClientRequest.On("SetHeaderParam", "Authorization", []string{"Bearer bearer123"}).Return(nil)

	authErr := client.Auth.AuthenticateRequest(mockClientRequest, nil)
	require.NoError(authErr)
	assert.True(mockClientRequest.AssertExpectations(t))
}

func TestSecretsAPI_InitializeClient_Success(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	apiSetting := api.GetSettings(api.ServiceSecrets)
	secretsapi.InitializeClient()

	client := secretsapi.Get()
	require.NotNil(client)
	assert.NotNil(client.Auth)
	assert.Equal(fmt.Sprintf("%s://%s%s", apiSetting.Schema, apiSetting.Host, apiSetting.BasePath), client.BaseURI)

	rt, isRuntime := client.Transport.(*httptransport.Runtime)
	require.True(isRuntime, "client.Transport is a Runtime")
	assert.Equal(apiSetting.Host, rt.Host)
	assert.Equal(apiSetting.BasePath, rt.BasePath)

	// validate that the client.Auth writer sets the bearer token using the one we provided
	mockClientRequest := new(MockClientRequest)
	mockClientRequest.On("SetHeaderParam", "Authorization", []string{"Bearer "}).Return(nil)

	authErr := client.Auth.AuthenticateRequest(mockClientRequest, nil)
	require.NoError(authErr)
	assert.True(mockClientRequest.AssertExpectations(t))
}

func TestSecretsAPI_Authenticated_Failure(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	client := secretsapi_test.NewDefaultTestClient("bearer123")
	require.NotNil(client)

	httpmock.Activate(client.BaseURI)
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("GET", "/whoami", 401)

	uid, failure := client.AuthenticatedUserID()
	assert.Zero(uid)
	assert.True(failure.Type.Matches(api.FailAuth), "should be an FailAuth failure")
}

func TestSecretsAPI_Authenticated_Success(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	client := secretsapi_test.NewDefaultTestClient("bearer123")
	require.NotNil(client)

	httpmock.Activate(client.BaseURI)
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("GET", "/whoami", 200)

	uid, failure := client.AuthenticatedUserID()
	assert.Nil(failure)
	assert.Equal("11110000-1111-0000-1111-000011110000", uid.String())
}
