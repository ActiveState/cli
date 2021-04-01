package authentication

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
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
	httpmock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)

	credentials := &mono_models.Credentials{
		Username: user.Username,
		Password: user.Password,
	}
	cfg, err := config.Get()
	require.NoError(t, err)
	auth := New(cfg)
	err = auth.AuthenticateWithModel(credentials)
	assert.NoError(t, err, "Can Authenticate")
	assert.NotEmpty(t, cfg.GetString("apiToken"), "Authentication is persisted through token")
	assert.True(t, auth.Authenticated(), "Authentication is persisted for this session")
	assert.Equal(t, "test", auth.WhoAmI(), "Should return username 'test'")

	Reset()
	auth = New(cfg)
	assert.True(t, auth.Authenticated(), "Authentication should still be valid")

	auth = New(cfg)
	err = auth.AuthenticateWithUser(credentials.Username, credentials.Password, "")
	assert.NoError(t, err, "Authentication should work again")
}

func TestAuthAPIKeyOverride(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")

	os.Setenv(constants.APIKeyEnvVarName, "testSuccess")
	defer os.Unsetenv(constants.APIKeyEnvVarName)
	cfg, err := config.Get()
	require.NoError(t, err)
	auth := New(cfg)
	err = auth.Authenticate()
	assert.NoError(t, err, "Authentication by user-defined token should not error")
	assert.True(t, auth.Authenticated(), "Authentication should still be valid")
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
	cfg, err := config.Get()
	require.NoError(t, err)
	auth := New(cfg)
	err = auth.AuthenticateWithModel(credentials)
	require.Error(t, err)
	assert.Equal(t, err.Error(), locale.T("err_unauthorized"), "Should fail to authenticate")
}

func TestAuthInvalidToken(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("POST", "/login", 401)

	cfg, err := config.Get()
	require.NoError(t, err)

	cfg.Set("apiToken", "testFailure")
	auth := New(cfg)
	err = auth.Authenticate()
	require.Error(t, err)
	assert.IsType(t, &ErrUnauthorized{}, err, "Should fail to authenticate")
	auth.Logout()
	assert.Empty(t, cfg.GetString("apiToken"), "", "apiToken should have cleared")
}

func TestClientFailure(t *testing.T) {
	cfg, err := config.Get()
	require.NoError(t, err)
	auth := New(cfg)
	var exitCode int
	exit = func(code int) {
		exitCode = code
	}
	auth.Client()
	assert.Equal(t, 1, exitCode, "Should exit")
}
