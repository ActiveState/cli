package auth_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
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
	failures.ResetHandled()
	authentication.Logout()
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

func TestUsernameValidator(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/users/uniqueUsername/test")

	err := authlet.UsernameValidator("test")
	assert.NoError(t, err, "Username is unique")

	httpmock.RegisterWithCode("GET", "/users/uniqueUsername/test", 400)

	err = authlet.UsernameValidator("test")
	assert.Error(t, err, "Username is not unique")
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

	pmock.OnMethod("Select").Once().Return(locale.T("prompt_login_action"), nil)
	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	authlet.RequireAuthentication("", outputhelper.NewCatcher(), pmock)

	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestRequireAuthenticationLoginFail(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/users/uniqueUsername/test")
	httpmock.RegisterWithCode("POST", "/login", 401)

	var fail *failures.Failure
	pmock.OnMethod("Select").Once().Return(locale.T("prompt_login_action"), nil)
	pmock.OnMethod("Input").Once().Return("Iammeanttofail", nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	fail = authlet.RequireAuthentication("", outputhelper.NewCatcher(), pmock)

	assert.Nil(t, authentication.ClientAuth(), "Not Authenticated")
	require.Error(t, fail.ToError(), "Failure occurred")
	assert.Equal(t, authlet.FailNotAuthenticated.Name, fail.Type.Name)
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
	authlet.RequireAuthentication("", outputhelper.NewCatcher(), pmock)

	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestRequireAuthenticationSignupBrowser(t *testing.T) {
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

	var openURICalled bool
	authlet.OpenURI = func(uri string) error {
		openURICalled = true
		return nil
	}

	pmock.OnMethod("Select").Once().Return(locale.T("prompt_signup_browser_action"), nil)
	pmock.OnMethod("Input").Once().Return("Iammeanttofail", nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	authlet.RequireAuthentication("", outputhelper.NewCatcher(), pmock)

	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
	assert.True(t, openURICalled, "OpenURI was called")
}
