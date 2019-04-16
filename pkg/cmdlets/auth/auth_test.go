package auth_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	promptMock "github.com/ActiveState/cli/internal/prompt/mock"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	authlet "github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/api"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	authCmd "github.com/ActiveState/cli/state/auth"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var Command = authCmd.Command

func setup(t *testing.T) {
	failures.ResetHandled()
	authentication.Logout()
	secretsapi_test.InitializeTestClient("bearer123")

	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})
	authCmd.Flags.Token = ""
	authCmd.Flags.Username = ""
	authCmd.Flags.Password = ""
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

func TestExecuteNoArgs(t *testing.T) {
	setup(t)
	pmock := promptMock.Init()
	authlet.Prompter = pmock
	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("POST", "/login", 401)

	pmock.OnMethod("Input").Once().Return("baduser", nil)
	pmock.OnMethod("InputSecret").Once().Return("badpass", nil)
	execErr := Command.Execute()

	assert.Error(t, execErr, "No failure occurred")
	assert.Nil(t, authentication.ClientAuth(), "Did not authenticate")
}

func TestExecuteNoArgsAuthenticated_WithExistingKeypair(t *testing.T) {
	setup(t)
	user := setupUser()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/renew")

	fail := authentication.Get().AuthenticateWithModel(&mono_models.Credentials{
		Username: user.Username,
		Password: user.Password,
	})
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	require.NoError(t, fail.ToError())

	assert.NoError(t, Command.Execute(), "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestExecuteNoArgsLoginByPrompt_WithExistingKeypair(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()
	authlet.Prompter = pmock

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	secretsapiMock := httpmock.Activate(secretsapi.Get().BaseURI)
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("POST", "/apikeys")
	secretsapiMock.Register("GET", "/keypair")

	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	execErr := Command.Execute()

	assert.NoError(t, execErr, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestExecuteNoArgsLoginByPrompt_NoExistingKeypair(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()
	authlet.Prompter = pmock

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	secretsapiMock := httpmock.Activate(secretsapi.Get().BaseURI)
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("POST", "/apikeys")

	var bodyKeypair *secretsModels.KeypairChange
	var bodyErr error
	secretsapiMock.RegisterWithCode("GET", "/keypair", 404)
	secretsapiMock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	execErr := Command.Execute()

	assert.NoError(t, execErr, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	require.NoError(t, bodyErr, "unmarshalling keypair save response")
	assert.NotZero(t, bodyKeypair.EncryptedPrivateKey, "published private key")
	assert.NotZero(t, bodyKeypair.PublicKey, "published public key")
}

func TestExecuteNoArgsLoginThenSignupByPrompt(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()
	authlet.Prompter = pmock

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	secretsapiMock := httpmock.Activate(secretsapi.Get().BaseURI)
	defer httpmock.DeActivate()

	var secondRequest bool
	httpmock.RegisterWithResponder("POST", "/login", func(req *http.Request) (int, string) {
		if !secondRequest {
			secondRequest = true
			return 401, "login"
		}
		return 200, "login"
	})
	httpmock.Register("GET", "/users/uniqueUsername/test")
	httpmock.Register("POST", "/users")

	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("POST", "/apikeys")

	var bodyKeypair *secretsModels.KeypairChange
	var bodyErr error
	secretsapiMock.RegisterWithCode("GET", "/keypair", 404)
	secretsapiMock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Twice().Return(user.Password, nil)
	pmock.OnMethod("Confirm").Once().Return(true, nil)
	pmock.OnMethod("Input").Once().Return(user.Email, nil)
	pmock.OnMethod("Input").Once().Return(user.Name, nil)
	execErr := Command.Execute()

	assert.NoError(t, execErr, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	require.NoError(t, bodyErr, "unmarshalling keypair save response")
	assert.NotZero(t, bodyKeypair.EncryptedPrivateKey, "published private key")
	assert.NotZero(t, bodyKeypair.PublicKey, "published public key")
}

func TestExecuteSignup(t *testing.T) {
	setup(t)
	pmock := promptMock.Init()
	authlet.Prompter = pmock

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	secretsapiMock := httpmock.Activate(secretsapi.Get().BaseURI)
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/users/uniqueUsername/test")
	httpmock.Register("POST", "/users")
	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("POST", "/apikeys")

	var bodyKeypair *secretsModels.KeypairChange
	var bodyErr error
	secretsapiMock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	user := setupUser()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"signup"})

	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Twice().Return(user.Password, nil)
	pmock.OnMethod("Input").Once().Return(user.Name, nil)
	pmock.OnMethod("Input").Once().Return(user.Email, nil)
	execErr := Command.Execute()

	assert.NoError(t, execErr, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	require.NoError(t, bodyErr, "unmarshalling keypair save response")
	assert.NotZero(t, bodyKeypair.EncryptedPrivateKey, "published private key")
	assert.NotZero(t, bodyKeypair.PublicKey, "published public key")
}

func TestExecuteToken(t *testing.T) {
	setup(t)
	user := setupUser()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("POST", "/apikeys")

	fail := authentication.Get().AuthenticateWithModel(&mono_models.Credentials{
		Username: user.Username,
		Password: user.Password,
	})
	token := viper.GetString("apiToken")
	authentication.Logout()
	assert.NoError(t, fail.ToError(), "Executed without error")
	assert.Nil(t, authentication.ClientAuth(), "Not Authenticated")

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--token", token})

	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestExecuteLogout(t *testing.T) {
	setup(t)
	defer osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	user := setupUser()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")

	a := authentication.Get()
	fail := a.AuthenticateWithModel(&mono_models.Credentials{
		Username: user.Username,
		Password: user.Password,
	})
	require.NoError(t, fail.ToError())
	assert.True(t, a.Authenticated(), "Authenticated")

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"logout"})

	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.False(t, a.Authenticated(), "Not Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	pkstat, err := osutil.StatConfigFile(constants.KeypairLocalFileName + ".key")
	require.Nil(t, pkstat)
	// Unux | Windows
	assert.Regexp(t, "[no such file or directory|The system cannot find the file specified]", err.Error())
}

func TestExecuteAuthWithTOTP_WithExistingKeypair(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()
	authlet.Prompter = pmock

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	secretsapiMock := httpmock.Activate(secretsapi.Get().BaseURI)
	defer httpmock.DeActivate()

	httpmock.RegisterWithResponder("POST", "/login", func(req *http.Request) (int, string) {
		bodyBytes, _ := ioutil.ReadAll(req.Body)
		bodyString := string(bodyBytes)
		if !strings.Contains(bodyString, "totp") {
			return 449, "login"
		}
		return 200, "login"
	})
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("POST", "/apikeys")
	secretsapiMock.Register("GET", "/keypair")

	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	pmock.OnMethod("Input").Once().Return("", nil)
	execErr := Command.Execute()

	require.NoError(t, execErr, "Executed without error")
	assert.Nil(t, authentication.ClientAuth(), "Not Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
	failures.ResetHandled()

	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	pmock.OnMethod("Input").Once().Return("foo", nil)
	execErr = Command.Execute()

	require.NoError(t, execErr, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
	failures.ResetHandled()
}

func TestExecuteAuthWithTOTP_NoExistingKeypair(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()
	authlet.Prompter = pmock

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	secretsapiMock := httpmock.Activate(secretsapi.Get().BaseURI)
	defer httpmock.DeActivate()
	defer failures.ResetHandled()

	httpmock.RegisterWithResponder("POST", "/login", func(req *http.Request) (int, string) {
		bodyBytes, _ := ioutil.ReadAll(req.Body)
		bodyString := string(bodyBytes)
		if !strings.Contains(bodyString, "totp") {
			return 449, "login"
		}
		return 200, "login"
	})
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("POST", "/apikeys")

	var bodyKeypair *secretsModels.KeypairChange
	var bodyErr error
	secretsapiMock.RegisterWithCode("GET", "/keypair", 404)
	secretsapiMock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	pmock.OnMethod("Input").Once().Return("", nil)
	execErr := Command.Execute()

	require.NoError(t, execErr, "Executed without error")
	assert.Nil(t, authentication.ClientAuth(), "Not Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
	failures.ResetHandled()

	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	pmock.OnMethod("Input").Once().Return("foo", nil)
	execErr = Command.Execute()

	require.NoError(t, execErr, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	require.NoError(t, bodyErr, "unmarshalling keypair save response")
	assert.NotZero(t, bodyKeypair.EncryptedPrivateKey, "published private key")
	assert.NotZero(t, bodyKeypair.PublicKey, "published public key")
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
	authlet.Prompter = pmock

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	secretsapiMock := httpmock.Activate(secretsapi.Get().BaseURI)
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/renew")
	secretsapiMock.Register("GET", "/keypair")

	pmock.OnMethod("Select").Once().Return(locale.T("prompt_login_action"), nil)
	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	authlet.RequireAuthentication("")

	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestRequireAuthenticationLoginFail(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()
	authlet.Prompter = pmock

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/users/uniqueUsername/test")
	httpmock.RegisterWithCode("POST", "/login", 401)

	var fail *failures.Failure
	pmock.OnMethod("Select").Once().Return(locale.T("prompt_login_action"), nil)
	pmock.OnMethod("Input").Once().Return("Iammeanttofail", nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	fail = authlet.RequireAuthentication("")

	assert.Nil(t, authentication.ClientAuth(), "Not Authenticated")
	require.Error(t, fail.ToError(), "Failure occurred")
	assert.Equal(t, authlet.FailNotAuthenticated.Name, fail.Type.Name)
}

func TestRequireAuthenticationSignup(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()
	authlet.Prompter = pmock

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	secretsapiMock := httpmock.Activate(secretsapi.Get().BaseURI)
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/users/uniqueUsername/test")
	httpmock.Register("POST", "/users")
	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("POST", "/apikeys")

	secretsapiMock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		return 204, "empty"
	})

	pmock.OnMethod("Select").Once().Return(locale.T("prompt_signup_action"), nil)
	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Twice().Return(user.Password, nil)
	pmock.OnMethod("Input").Once().Return(user.Name, nil)
	pmock.OnMethod("Input").Once().Return(user.Email, nil)
	authlet.RequireAuthentication("")

	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestRequireAuthenticationSignupBrowser(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()
	authlet.Prompter = pmock

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	secretsapiMock := httpmock.Activate(secretsapi.Get().BaseURI)
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
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
	authlet.RequireAuthentication("")

	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
	assert.True(t, openURICalled, "OpenURI was called")
}
