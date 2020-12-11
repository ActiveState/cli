package model_test

import (
	"net/url"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"

	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
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
	suite.apiMock.MockSignS3URI()

	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}

	u, _ := url.Parse("http://test.tld/python" + ext)
	response, err := model.SignS3URL(u)
	suite.Require().NoError(err)
	suite.Equal(u.String(), response.String())
}

func TestS3Suite(t *testing.T) {
	suite.Run(t, new(S3TestSuite))
}
