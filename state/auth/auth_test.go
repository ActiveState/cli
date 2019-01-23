package auth

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) {
	api.RemoveAuth()
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

	testCredentials = &models.Credentials{}
	testSignupInput = &signupInput{}

	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
	assert.Nil(t, api.Auth, "Did not authenticate")
}

func TestExecuteNoArgsAuthenticated(t *testing.T) {
	setup(t)
	user := setupUser(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("GET", "/renew")

	testCredentials = &models.Credentials{
		Username: user.Username,
		Password: user.Password,
	}
	_, err := api.Authenticate(testCredentials)
	assert.NotNil(t, api.Auth, "Authenticated")

	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestExecuteNoArgsLoginByPrompt(t *testing.T) {
	setup(t)
	user := setupUser(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("GET", "/renew")

	testCredentials = &models.Credentials{
		Username: user.Username,
		Password: user.Password,
	}

	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NotNil(t, api.Auth, "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestExecuteNoArgsLoginThenSignupByPrompt(t *testing.T) {
	setup(t)
	user := setupUser(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	var secondRequest bool
	httpmock.RegisterWithResponder("POST", "/login", func(req *http.Request) (int, string) {
		if !secondRequest {
			secondRequest = true
			return 401, "login"
		}
		return 200, "login"
	})
	httpmock.Register("POST", "/users")
	httpmock.Register("GET", "/users/uniqueUsername/test")
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("GET", "/renew")

	testCredentials = &models.Credentials{
		Username: user.Username,
		Password: user.Password,
	}

	testSignupInput = &signupInput{
		Name:      user.Name,
		Email:     user.Email,
		Username:  user.Username,
		Password:  user.Password,
		Password2: user.Password,
	}

	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NotNil(t, api.Auth, "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestExecuteSignup(t *testing.T) {
	setup(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("POST", "/users")
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("GET", "/renew")

	user := setupUser(t)

	testSignupInput = &signupInput{
		Name:      user.Name,
		Email:     user.Email,
		Username:  user.Username,
		Password:  user.Password,
		Password2: user.Password,
	}

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"signup"})

	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NotNil(t, api.Auth, "Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestExecuteToken(t *testing.T) {
	setup(t)
	user := setupUser(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("GET", "/renew")

	testCredentials = &models.Credentials{
		Username: user.Username,
		Password: user.Password,
	}
	_, err := api.Authenticate(testCredentials)
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
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	user := setupUser(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("GET", "/renew")

	testCredentials = &models.Credentials{
		Username: user.Username,
		Password: user.Password,
	}
	_, err := api.Authenticate(testCredentials)
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
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)
	httpmock.Register("GET", "/renew")

	testCredentials = &models.Credentials{
		Username: user.Username,
		Password: user.Password,
	}

	logging.Debug("Executing..")
	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.Nil(t, api.Auth, "Not Authenticated")
	assert.NoError(t, failures.Handled(), "No failure occurred")
	failures.ResetHandled()

	testCredentials.Totp = "foo"
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
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
