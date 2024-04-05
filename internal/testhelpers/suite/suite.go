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
	IsHelperSuite() bool
}

type Suite struct {
	testify.Suite
}

func (suite *Suite) NoError(err error, msgAndArgs ...interface{}) {
	if err == nil {
		return
	}

	suite.Fail(fmt.Sprintf("Received unexpected error:\n%+v", errs.JoinMessage(err)), msgAndArgs...)
}

func (suite *Suite) Require() *Assertions {
	return &Assertions{suite.Suite.Require()}
}

func (suite *Suite) IsHelperSuite() bool {
	return true
}

type Assertions struct {
	*require.Assertions
}

func (a *Assertions) NoError(err error, msgAndArgs ...interface{}) {
	if err == nil {
		return
	}

	a.Fail(fmt.Sprintf("Received unexpected error:\n%+v", errs.JoinMessage(err)), msgAndArgs...)
}

func Run(t *testing.T, suite tSuite) {
	testify.Run(t, suite)
}
