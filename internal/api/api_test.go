package api

import (
	"testing"

	"github.com/ActiveState/cli/internal/api/client/users"
	"github.com/stretchr/testify/assert"
)

func TestApi(t *testing.T) {
	// We're just testing an easy to use API endpoint here, the point of this test is to test the lib, not the endpoint
	params := users.NewUniqueUsernameParams()
	params.SetUsername("DontCreateAUserWIthThisName")
	res, err := Client.Users.UniqueUsername(params)
	assert.NoError(t, err)
	assert.Equal(t, int64(200), *res.Payload.Code, "Should return HTTP Code 200")
}
