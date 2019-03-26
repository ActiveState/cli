package authentication

import (
	"testing"

	clientAuth "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/authentication"

	"github.com/ActiveState/cli/pkg/platform/api"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) {
	Logout()
}

func setupUser(t *testing.T) *mono_models.UserEditable {
	return &mono_models.UserEditable{
		Username: "test",
		Email:    "test@test.tld",
		Password: "test",
		Name:     "test",
	}
}

func TestAuth(t *testing.T) {
	setup(t)
	user := setupUser(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)

	credentials := &mono_models.Credentials{
		Username: user.Username,
		Password: user.Password,
	}
	auth := New()
	fail := auth.AuthenticateWithModel(credentials)
	assert.NoError(t, fail.ToError(), "Can Authenticate")
	assert.NotEmpty(t, viper.GetString("apiToken"), "Authentication is persisted through token")
	assert.True(t, auth.Authenticated(), "Authentication is persisted for this session")
	assert.Equal(t, "test", auth.WhoAmI(), "Should return username 'test'")

	Reset()
	auth = New()
	assert.NotNil(t, auth.Authenticated(), "Authentication is still persisted for this session")

	auth = New()
	fail = auth.AuthenticateWithUser(credentials.Username, credentials.Password, "")
	assert.NoError(t, fail.ToError(), "Can Authenticate Again")
}

func TestPersist(t *testing.T) {
	auth := Get()
	auth2 := Get()
	assert.True(t, auth == auth2, "Should return same pointer")
}

func TestAuthInvalidUser(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("POST", "/login", 401)

	credentials := &mono_models.Credentials{
		Username: "testFailure",
		Password: "testFailure",
	}
	auth := New()
	fail := auth.AuthenticateWithModel(credentials)
	assert.IsType(t, new(clientAuth.PostLoginUnauthorized), fail.ToError(), "Should fail to authenticate")
}

func TestAuthInvalidToken(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("POST", "/login", 401)

	viper.Set("apiToken", "testFailure")
	auth := New()
	fail := auth.Authenticate()
	assert.Error(t, fail.ToError(), "Should not have authenticated")
	assert.Empty(t, viper.GetString("apiToken"), "", "apiToken should have cleared")
}

func TestClientFailure(t *testing.T) {
	auth := New()
	var exitCode int
	exit = func(code int) {
		exitCode = code
	}
	auth.Client()
	assert.Equal(t, 1, exitCode, "Should exit")
}
