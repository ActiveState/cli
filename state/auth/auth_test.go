package auth

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
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
)

func setup(t *testing.T) {
	failures.ResetHandled()
	authentication.Logout()
	secretsapi_test.InitializeTestClient("bearer123")

	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})
	Flags.Token = ""
	Flags.Username = ""
	Flags.Password = ""
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

func TestExecuteNoArgsAuthenticated(t *testing.T) {
	setup(t)
	user := setupUser()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/apikeys")
	httpmock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/renew")

	secretMock := httpmock.Activate(api.GetServiceURL(api.ServiceSecrets).String())
	secretMock.Register("GET", "/keypair")

	fail := authentication.Get().AuthenticateWithModel(&mono_models.Credentials{
		Username: user.Username,
		Password: user.Password,
	})
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	require.NoError(t, fail.ToError())

	assert.NoError(t, Command.Execute(), "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestExecuteAuthenticatedByPrompts(t *testing.T) {
	setup(t)
	user := setupUser()
	pmock := promptMock.Init()
	authlet.Prompter = pmock

	monoMock := httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	monoMock.Register("POST", "/login")
	monoMock.Register("GET", "/apikeys")
	monoMock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
	monoMock.Register("POST", "/apikeys")
	monoMock.Register("GET", "/renew")

	secretMock := httpmock.Activate(api.GetServiceURL(api.ServiceSecrets).String())
	secretMock.Register("GET", "/keypair")

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})

	var execErr error
	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Once().Return(user.Password, nil)
	execErr = Command.Execute()

	assert.NoError(t, execErr, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
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

	secretMock := httpmock.Activate(api.GetServiceURL(api.ServiceSecrets).String())
	secretMock.Register("GET", "/keypair")

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--username", user.Username, "--password", user.Password})
	err := Command.Execute()

	assert.NoError(t, err, "Executed without error")
	assert.NotNil(t, authentication.ClientAuth(), "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
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
	httpmock.RegisterWithResposne("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
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

	var execErr error
	pmock.OnMethod("Input").Once().Return(user.Username, nil)
	pmock.OnMethod("InputSecret").Twice().Return(user.Password, nil)
	pmock.OnMethod("Input").Once().Return(user.Name, nil)
	pmock.OnMethod("Input").Once().Return(user.Email, nil)
	execErr = Command.Execute()

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
	httpmock.RegisterWithResponse("DELETE", "/apikeys/"+constants.APITokenName, 200, "/apikeys/"+constants.APITokenNamePrefix)
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

	auth := authentication.Get()
	fail := auth.AuthenticateWithModel(&mono_models.Credentials{
		Username: user.Username,
		Password: user.Password,
	})
	require.NoError(t, fail.ToError())
	assert.True(t, auth.Authenticated(), "Authenticated")

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"logout"})

	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.False(t, auth.Authenticated(), "Not Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	pkstat, err := osutil.StatConfigFile(constants.KeypairLocalFileName + ".key")
	require.Nil(t, pkstat)
	if runtime.GOOS != "windows" {
		assert.Regexp(t, "no such file or directory", err.Error())
	} else {
		assert.Regexp(t, "The system cannot find the file specified", err.Error())

	}
}
