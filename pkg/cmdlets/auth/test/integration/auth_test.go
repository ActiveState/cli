package integration_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/locale"
	promptMock "github.com/ActiveState/cli/internal/prompt/mock"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func setup(t *testing.T) {
	cfg, err := config.New()
	assert.NoError(t, err)
	auth := authentication.New(cfg)
	assert.NoError(t, auth.Logout())

	secretsapi_test.InitializeTestClient("bearer123")

	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	authlet.OpenURI = func(uri string) error { return nil }
}

func setupUser() *mono_models.UserEditable {
	testUser := &mono_models.UserEditable{
		Username: "test",
		Email:    "test@test.tld",
		Password: "foo", // this matches the passphrase on testdata/self-private.key
		Name:     "Test User",
	}
	return testUser
}

func TestRequireAuthenticationLogin(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	secretsapiMock := httpmock.Activate(secretsapi.Get().BaseURI)
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/apikeys")
	httpmock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/renew")
	secretsapiMock.Register("GET", "/keypair")

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()

	pmock.OnMethod("Select").Once().Return(locale.T("prompt_login_action"), nil)
	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)

	auth := authentication.New(cfg)
	defer auth.Close()

	authlet.RequireAuthentication("", cfg, outputhelper.NewCatcher(), pmock, auth)

	assert.NotNil(t, auth.ClientAuth(), "Authenticated")
}

func TestRequireAuthenticationLoginFail(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/users/uniqueUsername/test")
	httpmock.RegisterWithCode("POST", "/login", 401)

	var err error
	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	pmock.OnMethod("Select").Once().Return(locale.T("prompt_login_action"), nil)
	pmock.OnMethod("Input").Once().Return("Iammeanttoerr", nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	auth := authentication.New(cfg)
	defer auth.Close()
	err = authlet.RequireAuthentication("", cfg, outputhelper.NewCatcher(), pmock, auth)

	assert.Nil(t, auth.ClientAuth(), "Not Authenticated")
	require.Error(t, err, "Failure occurred")
}

func TestRequireAuthenticationSignup(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	secretsapiMock := httpmock.Activate(secretsapi.Get().BaseURI)
	asMock := httpmock.Activate("https://www.activestate.com")
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/users/uniqueUsername/test")
	httpmock.Register("POST", "/users")
	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/apikeys")
	httpmock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	httpmock.Register("POST", "/apikeys")
	asMock.RegisterWithResponseBody("GET", strings.TrimPrefix(constants.TermsOfServiceURLText, "https://www.activestate.com"), 200, "")

	secretsapiMock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		return 204, "empty"
	})

	pmock.OnMethod("Select").Once().Return(locale.T("prompt_signup_action"), nil)
	pmock.OnMethod("Select").Once().Return(locale.T("tos_accept"), nil)
	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Twice().Return(user.Password, nil)
	pmock.OnMethod("Input").Once().Return(user.Name, nil)
	pmock.OnMethod("Input").Once().Return(user.Email, nil)
	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	auth := authentication.New(cfg)
	defer auth.Close()
	authlet.RequireAuthentication("", cfg, outputhelper.NewCatcher(), pmock, auth)

	assert.NotNil(t, auth.ClientAuth(), "Authenticated")
}

func TestRequireAuthenticationSignupBrowser(t *testing.T) {
	setup(t)
	pmock := promptMock.Init()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	secretsapiMock := httpmock.Activate(secretsapi.Get().BaseURI)
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/apikeys")
	httpmock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("POST", "/oauth/authorize/device")
	httpmock.Register("GET", "/oauth/authorize/device")
	secretsapiMock.Register("GET", "/keypair")

	var openURICalled bool
	authlet.OpenURI = func(uri string) error {
		openURICalled = true
		return nil
	}

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	pmock.OnMethod("Select").Once().Return(locale.T("prompt_signup_browser_action"), nil)
	auth := authentication.New(cfg)
	defer auth.Close()
	authlet.RequireAuthentication("", cfg, outputhelper.NewCatcher(), pmock, auth)

	assert.True(t, openURICalled, "OpenURI was called")
	assert.NotNil(t, auth.ClientAuth(), "Authenticated")
}

func TestRequireAuthenticationSignupBrowserTimeout(t *testing.T) {
	setup(t)
	pmock := promptMock.Init()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	secretsapiMock := httpmock.Activate(secretsapi.Get().BaseURI)
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/apikeys")
	httpmock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("POST", "/oauth/authorize/device")
	httpmock.RegisterWithCode("GET", "/oauth/authorize/device", 400)
	secretsapiMock.Register("GET", "/keypair")

	var openURICalled bool
	authlet.OpenURI = func(uri string) error {
		openURICalled = true
		return nil
	}

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	pmock.OnMethod("Select").Once().Return(locale.T("prompt_signup_browser_action"), nil)
	auth := authentication.New(cfg)
	defer auth.Close()
	err = authlet.RequireAuthentication("", cfg, outputhelper.NewCatcher(), pmock, auth)

	assert.True(t, openURICalled, "OpenURI was called")
	assert.Nil(t, auth.ClientAuth(), "Not Authenticated")
	require.Error(t, err, "Failure occurred")
}

func TestRequireAuthenticationLoginBrowser(t *testing.T) {
	setup(t)
	pmock := promptMock.Init()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	secretsapiMock := httpmock.Activate(secretsapi.Get().BaseURI)
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/apikeys")
	httpmock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("POST", "/oauth/authorize/device")
	httpmock.Register("GET", "/oauth/authorize/device")
	secretsapiMock.Register("GET", "/keypair")

	var openURICalled bool
	authlet.OpenURI = func(uri string) error {
		openURICalled = true
		return nil
	}

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	pmock.OnMethod("Select").Once().Return(locale.T("prompt_login_browser_action"), nil)
	auth := authentication.New(cfg)
	defer auth.Close()
	authlet.RequireAuthentication("", cfg, outputhelper.NewCatcher(), pmock, auth)

	assert.NotNil(t, auth.ClientAuth(), "Authenticated")
	assert.True(t, openURICalled, "OpenURI was called")
}
