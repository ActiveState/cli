package model_test

import (
	"net/url"
	"testing"

	apiMock "github.com/ActiveState/cli/pkg/platform/api/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/stretchr/testify/suite"
)

type S3TestSuite struct {
	suite.Suite
	apiMock  *apiMock.Mock
	authMock *authMock.Mock
}

func (suite *S3TestSuite) BeforeTest(suiteName, testName string) {
	suite.apiMock = apiMock.Init()
	suite.authMock = authMock.Init()
}

func (suite *S3TestSuite) AfterTest(suiteName, testName string) {
	suite.apiMock.Close()
	suite.authMock.Close()
}

func (suite *S3TestSuite) TestGetS3() {
	suite.authMock.MockLoggedin()
	suite.apiMock.MockSignS3URI()

	u, _ := url.Parse("http://test.tld/archive.tar.gz")
	response, fail := model.SignS3URL(u)
	suite.Require().NoError(fail.ToError())
	suite.Equal(u.String(), response.String())
}

func TestS3Suite(t *testing.T) {
	suite.Run(t, new(S3TestSuite))
}
