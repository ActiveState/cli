package auth

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) {
	failures.ResetHandled()
	api.RemoveAuth()
	secretsapi_test.InitializeTestClient("bearer123")

	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})
	Args.Token = ""
}

func setupUser(t *testing.T) *models.UserEditable {
	testUser := &models.UserEditable{
		Username: "test",
		Email:    "test@test.tld",
		Password: "test",
		Name:     "test",
	}
	return testUser
}

func TestExecuteNoArgs(t *testing.T) {
	setup(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("POST", "/login", 401)

	var execErr error
	osutil.WrapStdinWithDelay(10*time.Millisecond, func() { execErr = Command.Execute() },
		// prompted for username and password only
		// 10ms delay between writes to stdin
		"baduser",
		"badpass",
	)

	assert.NoError(t, execErr, "Executed without error")
	assert.Error(t, failures.Handled(), "No failure occurred")
	assert.Nil(t, api.Auth, "Did not authenticate")
}

func TestExecuteNoArgsAuthenticated(t *testing.T) {
	setup(t)
	user := setupUser(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/renew")

	_, err := api.Authenticate(&models.Credentials{
		Username: user.Username,
		Password: user.Password,
	})
	assert.NotNil(t, api.Auth, "Authenticated")
	require.NoError(t, err)

	assert.NoError(t, Command.Execute(), "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestExecuteNoArgsLoginByPrompt(t *testing.T) {
	setup(t)
	user := setupUser(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")

	var execErr error
	osutil.WrapStdinWithDelay(10*time.Millisecond, func() { execErr = Command.Execute() },
		user.Username,
		user.Password)

	assert.NoError(t, execErr, "Executed without error")
	assert.NotNil(t, api.Auth, "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestExecuteNoArgsLoginThenSignupByPrompt(t *testing.T) {
	setup(t)
	user := setupUser(t)

	httpmock.Activate(api.Prefix)
	secretsapiMock := httpmock.Activate(secretsapi.DefaultClient.BaseURI)
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

	var bodyKeypair *secretsModels.KeypairChange
	var bodyErr error
	secretsapiMock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	var execErr error
	osutil.WrapStdinWithDelay(10*time.Millisecond, func() { execErr = Command.Execute() },
		// prompted for username and password
		user.Username,
		user.Password,
		// prompted to signup instead
		"yes",
		// enter new user details
		user.Password, // confirmation
		user.Name,
		user.Email,
	)

	assert.NoError(t, execErr, "Executed without error")
	assert.NotNil(t, api.Auth, "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	require.NoError(t, bodyErr, "unmarshalling keypair save response")
	assert.NotZero(t, bodyKeypair.EncryptedPrivateKey, "published private key")
	assert.NotZero(t, bodyKeypair.PublicKey, "published public key")
}

func TestExecuteSignup(t *testing.T) {
	setup(t)

	httpmock.Activate(api.Prefix)
	secretsapiMock := httpmock.Activate(secretsapi.DefaultClient.BaseURI)
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/users/uniqueUsername/test")
	httpmock.Register("POST", "/users")
	httpmock.Register("POST", "/login")

	var bodyKeypair *secretsModels.KeypairChange
	var bodyErr error
	secretsapiMock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "empty"
	})

	user := setupUser(t)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"signup"})

	var execErr error
	osutil.WrapStdinWithDelay(10*time.Millisecond, func() { execErr = Command.Execute() },
		user.Username,
		user.Password,
		user.Password, // confirmation
		user.Name,
		user.Email,
	)

	assert.NoError(t, execErr, "Executed without error")
	assert.NotNil(t, api.Auth, "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	require.NoError(t, bodyErr, "unmarshalling keypair save response")
	assert.NotZero(t, bodyKeypair.EncryptedPrivateKey, "published private key")
	assert.NotZero(t, bodyKeypair.PublicKey, "published public key")
}

func TestExecuteToken(t *testing.T) {
	setup(t)
	user := setupUser(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("POST", "/apikeys")

	_, err := api.Authenticate(&models.Credentials{
		Username: user.Username,
		Password: user.Password,
	})
	token := viper.GetString("apiToken")
	api.RemoveAuth()
	assert.NoError(t, err, "Executed without error")
	assert.Nil(t, api.Auth, "Not Authenticated")

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{token})

	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NotNil(t, api.Auth, "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestExecuteLogout(t *testing.T) {
	setup(t)
	defer osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	user := setupUser(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")

	_, err := api.Authenticate(&models.Credentials{
		Username: user.Username,
		Password: user.Password,
	})
	assert.NotNil(t, api.Auth, "Authenticated")

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"logout"})

	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.Nil(t, api.Auth, "Not Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	pkstat, err := osutil.StatConfigFile(constants.KeypairLocalFileName + ".key")
	require.Nil(t, pkstat)
	assert.Regexp(t, "no such file or directory", err.Error())
}

func TestExecuteAuthWithTOTP(t *testing.T) {
	setup(t)
	user := setupUser(t)

	httpmock.Activate(api.Prefix)
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

	var execErr error
	// \x04 is the equivalent of a ctrl+d, which tells the survey prompter to stop expecting
	// input for the specific field
	osutil.WrapStdinWithDelay(10*time.Millisecond,
		func() { execErr = Command.Execute() },
		user.Username, user.Password, "\x04")

	require.NoError(t, execErr, "Executed without error")
	assert.Nil(t, api.Auth, "Not Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
	failures.ResetHandled()

	osutil.WrapStdinWithDelay(10*time.Millisecond,
		func() { execErr = Command.Execute() },
		user.Username, user.Password, "foo")

	require.NoError(t, execErr, "Executed without error")
	assert.NotNil(t, api.Auth, "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
	failures.ResetHandled()
}

func TestUsernameValidator(t *testing.T) {
	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/users/uniqueUsername/test")

	err := usernameValidator("test")
	assert.NoError(t, err, "Username is unique")

	httpmock.RegisterWithCode("GET", "/users/uniqueUsername/test", 400)

	err = usernameValidator("test")
	assert.Error(t, err, "Username is not unique")
}
