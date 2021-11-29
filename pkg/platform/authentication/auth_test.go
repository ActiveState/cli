package authentication

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/config"
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

func TestPersist(t *testing.T) {
	auth := LegacyGet()
	auth2 := LegacyGet()
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
	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	auth := New(cfg)
	err = auth.AuthenticateWithModel(credentials)
	require.Error(t, err)
	assert.Equal(t, err.Error(), locale.T("err_unauthorized"), "Should fail to authenticate")
}

func TestClientFailure(t *testing.T) {
	cfg, err := config.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, cfg.Close()) }()
	auth := New(cfg)
	var exitCode int
	exit = func(code int) {
		exitCode = code
	}
	auth.Client()
	assert.Equal(t, 1, exitCode, "Should exit")
}
