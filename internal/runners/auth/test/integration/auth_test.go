package integration

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/runners/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/prompt"
	promptMock "github.com/ActiveState/cli/internal/prompt/mock"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func setup(t *testing.T) {
	authentication.Logout()
	secretsapi_test.InitializeTestClient("bearer123")

	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
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

func runAuth(params *auth.AuthParams, prompter prompt.Prompter, cfg keypairs.Configurable) error {
	auth := &auth.Auth{outputhelper.NewCatcher(), authentication.LegacyGet(), prompter, cfg}
	return auth.Run(params)
}

func runSignup(prompter prompt.Prompter, cfg keypairs.Configurable) error {
	signup := &auth.Signup{outputhelper.NewCatcher(), prompter, cfg}
	return signup.Run()
}

func runLogout(cfg keypairs.Configurable) error {
	signup := &auth.Logout{outputhelper.NewCatcher(), authentication.LegacyGet(), cfg}
	return signup.Run()
}

func TestExecuteNoArgsAuthenticated(t *testing.T) {
	user := setupUser()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/apikeys")
	httpmock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/renew")
	httpmock.Register("GET", "/tiers")
	httpmock.Register("GET", "/organizations/test")

	secretMock := httpmock.Activate(api.GetServiceURL(api.ServiceSecrets).String())
	secretMock.Register("GET", "/keypair")

	err := authentication.LegacyGet().AuthenticateWithModel(&mono_models.Credentials{
		Username: user.Username,
		Password: user.Password,
	})
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	require.NoError(t, err)

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()

	assert.NoError(t, runAuth(&auth.AuthParams{}, nil, cfg), "Executed without error")
}

func TestExecuteNoArgsNotAuthenticated(t *testing.T) {
	setup(t)
	pmock := promptMock.Init()
	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("POST", "/login", 401)

	pmock.OnMethod("Input").Once().Return("baduser", nil)
	pmock.OnMethod("InputSecret").Once().Return("badpass", nil)
	pmock.OnMethod("Input").Once().Return("foo", nil)

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()

	err = runAuth(&auth.AuthParams{}, pmock, cfg)
	assert.Error(t, err)
	assert.Nil(t, authentication.ClientAuth(), "Did not authenticate")
}

func TestExecuteNoArgsAuthenticated_WithExistingKeypair(t *testing.T) {
	setup(t)
	user := setupUser()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/apikeys")
	httpmock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/renew")
	httpmock.Register("GET", "/tiers")
	httpmock.Register("GET", "/organizations/test")

	err := authentication.LegacyGet().AuthenticateWithModel(&mono_models.Credentials{
		Username: user.Username,
		Password: user.Password,
	})
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	require.NoError(t, err)

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()

	assert.NoError(t, runAuth(&auth.AuthParams{}, nil, cfg), "Executed without error")
}

func TestExecuteNoArgsLoginByPrompt_WithExistingKeypair(t *testing.T) {
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
	secretsapiMock.Register("GET", "/keypair")
	httpmock.Register("GET", "/tiers")
	httpmock.Register("GET", "/organizations/test")

	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	err = runAuth(&auth.AuthParams{}, pmock, cfg)

	assert.NoError(t, err, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
}

func TestExecuteNoArgsLoginByPrompt_NoExistingKeypair(t *testing.T) {
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
	httpmock.Register("GET", "/tiers")
	httpmock.Register("GET", "/organizations/test")

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
	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	err = runAuth(&auth.AuthParams{}, pmock, cfg)

	assert.NoError(t, err, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")

	require.NoError(t, bodyErr, "unmarshalling keypair save response")
	assert.NotZero(t, bodyKeypair.EncryptedPrivateKey, "published private key")
	assert.NotZero(t, bodyKeypair.PublicKey, "published public key")
}

func TestExecuteNoArgsLoginThenSignupByPrompt(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()

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
	httpmock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/tiers")
	httpmock.Register("GET", "/organizations/test")

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
	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	err = runAuth(&auth.AuthParams{}, pmock, cfg)

	assert.NoError(t, err, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")

	require.NoError(t, bodyErr, "unmarshalling keypair save response")
	assert.NotZero(t, bodyKeypair.EncryptedPrivateKey, "published private key")
	assert.NotZero(t, bodyKeypair.PublicKey, "published public key")
}

func TestExecuteAuthenticatedByPrompts(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()

	monoMock := httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	monoMock.Register("POST", "/login")
	monoMock.Register("GET", "/apikeys")
	monoMock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	monoMock.Register("POST", "/apikeys")
	monoMock.Register("GET", "/renew")
	httpmock.Register("GET", "/tiers")
	httpmock.Register("GET", "/organizations/test")

	secretMock := httpmock.Activate(api.GetServiceURL(api.ServiceSecrets).String())
	secretMock.Register("GET", "/keypair")

	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	err = runAuth(&auth.AuthParams{}, pmock, cfg)

	assert.NoError(t, err, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
}

func TestExecuteAuthenticatedByFlags(t *testing.T) {
	setup(t)
	user := setupUser()

	monoMock := httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	monoMock.Register("POST", "/login")
	monoMock.Register("GET", "/apikeys")
	monoMock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	monoMock.Register("POST", "/apikeys")
	monoMock.Register("GET", "/renew")
	monoMock.Register("GET", "/tiers")
	monoMock.Register("GET", "/organizations/test")

	secretMock := httpmock.Activate(api.GetServiceURL(api.ServiceSecrets).String())
	secretMock.Register("GET", "/keypair")

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	err = runAuth(&auth.AuthParams{
		Username: user.Username,
		Password: user.Password,
	}, nil, cfg)

	assert.NoError(t, err, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
}

func TestExecuteSignup(t *testing.T) {
	setup(t)
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
	httpmock.Register("GET", "/tiers")
	httpmock.Register("GET", "/organizations/test")
	asMock.RegisterWithResponseBody("GET", strings.TrimPrefix(constants.TermsOfServiceURLText, "https://www.activestate.com"), 200, "")

	var bodyKeypair *secretsModels.KeypairChange
	var bodyErr error
	secretsapiMock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	user := setupUser()

	pmock.OnMethod("Select").Once().Return(locale.T("tos_accept"), nil)
	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Twice().Return(user.Password, nil)
	pmock.OnMethod("Input").Once().Return(user.Name, nil)
	pmock.OnMethod("Input").Once().Return(user.Email, nil)
	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	err = runSignup(pmock, cfg)

	assert.NoError(t, err, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")

	require.NoError(t, bodyErr, "unmarshalling keypair save response")
	assert.NotZero(t, bodyKeypair.EncryptedPrivateKey, "published private key")
	assert.NotZero(t, bodyKeypair.PublicKey, "published public key")
}

func TestExecuteSignup_DenyTOS(t *testing.T) {
	setup(t)
	pmock := promptMock.Init()

	pmock.OnMethod("Select").Once().Return(locale.T("tos_not_accept"), nil)

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	err = runSignup(pmock, cfg)
	assert.Error(t, err, "Executed with error")
}

func TestExecuteToken(t *testing.T) {
	setup(t)
	user := setupUser()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/apikeys")
	httpmock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/tiers")
	httpmock.Register("GET", "/organizations/test")

	err := authentication.LegacyGet().AuthenticateWithModel(&mono_models.Credentials{
		Username: user.Username,
		Password: user.Password,
	})
	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	token := cfg.GetString("apiToken")
	authentication.Logout()
	assert.NoError(t, err, "Executed without error")
	assert.Nil(t, authentication.ClientAuth(), "Not Authenticated")

	cfg2, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg2.Close()) }()
	err = runAuth(&auth.AuthParams{Token: token}, nil, cfg)

	assert.NoError(t, err, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
}

func TestExecuteLogout(t *testing.T) {
	setup(t)
	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	defer osutil.RemoveConfigFile(cfg.ConfigPath(), constants.KeypairLocalFileName+".key")
	osutil.CopyTestFileToConfigDir(cfg.ConfigPath(), "self-private.key", constants.KeypairLocalFileName+".key", 0600)

	user := setupUser()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("POST", "/apikeys")

	auth := authentication.LegacyGet()
	err = auth.AuthenticateWithModel(&mono_models.Credentials{
		Username: user.Username,
		Password: user.Password,
	})
	require.NoError(t, err, errs.Join(err, "\n").Error())
	assert.True(t, auth.Authenticated(), "Authenticated")

	err = runLogout(cfg)
	assert.NoError(t, err, "Executed without error")
	assert.False(t, auth.Authenticated(), "Not Authenticated")

	pkstat, err := osutil.StatConfigFile(cfg.ConfigPath(), constants.KeypairLocalFileName+".key")
	require.Nil(t, pkstat)
	if runtime.GOOS != "windows" {
		assert.Regexp(t, "no such file or directory", err.Error())
	} else {
		assert.Regexp(t, "The system cannot find the file specified", err.Error())

	}
}

func TestExecuteAuthWithTOTP_WithExistingKeypair(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()

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
	httpmock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/tiers")
	httpmock.Register("GET", "/organizations/test")
	secretsapiMock.Register("GET", "/keypair")

	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	pmock.OnMethod("Input").Once().Return("", nil)

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	err = runAuth(&auth.AuthParams{}, pmock, cfg)
	assert.Error(t, err)
	assert.Nil(t, authentication.ClientAuth(), "Not Authenticated")

	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	pmock.OnMethod("Input").Once().Return("foo", nil)
	err = runAuth(&auth.AuthParams{}, pmock, cfg)

	require.NoError(t, err, errs.Join(err, "\n").Error())
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
}

func TestExecuteAuthWithTOTP_NoExistingKeypair(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()

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
	httpmock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/tiers")
	httpmock.Register("GET", "/organizations/test")

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

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	err = runAuth(&auth.AuthParams{}, pmock, cfg)
	assert.Error(t, err)
	assert.Nil(t, authentication.ClientAuth(), "Not Authenticated")

	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	pmock.OnMethod("Input").Once().Return("foo", nil)
	err = runAuth(&auth.AuthParams{}, pmock, cfg)

	require.NoError(t, err, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")

	require.NoError(t, bodyErr, "unmarshalling keypair save response")
	assert.NotZero(t, bodyKeypair.EncryptedPrivateKey, "published private key")
	assert.NotZero(t, bodyKeypair.PublicKey, "published public key")
}

func TestExecuteWithTOTPFlag(t *testing.T) {
	setup(t)
	user := setupUser()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/tiers")
	httpmock.Register("GET", "/organizations/test")
	secretMock := httpmock.Activate(api.GetServiceURL(api.ServiceSecrets).String())
	secretMock.Register("GET", "/keypair")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("POST", "/apikeys")

	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	err = runAuth(&auth.AuthParams{
		Username: user.Username,
		Password: user.Password,
		Totp:     "123456",
	}, nil, cfg)
	require.NoError(t, err, errs.Join(err, "\n").Error())
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
}
