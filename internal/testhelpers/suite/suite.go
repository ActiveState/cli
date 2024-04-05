package suite

import (
	"fmt"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/stretchr/testify/require"
	testify "github.com/stretchr/testify/suite"
)

type tSuite interface {
	testify.TestingSuite
}

type Suite struct {
	testify.Suite
}

func (suite *Suite) NoError(err error, msgAndArgs ...interface{}) {
	if err == nil {
		return
	}

	errMsg := fmt.Sprintf("All error messages: %s", errs.JoinMessage(err))
	msgAndArgs = append(msgAndArgs, errMsg)
	suite.Suite.NoError(err, msgAndArgs...)
}

func (suite *Suite) Require() *Assertions {
	return &Assertions{suite.Suite.Require()}
}

type Assertions struct {
	*require.Assertions
}

func (a *Assertions) NoError(err error, msgAndArgs ...interface{}) {
	if err == nil {
		return
	}

	errMsg := fmt.Sprintf("All error messages: %s", errs.JoinMessage(err))
	msgAndArgs = append(msgAndArgs, errMsg)
	a.Assertions.NoError(err, msgAndArgs...)
}

func Run(t *testing.T, suite tSuite) {
	testify.Run(t, suite)
}
