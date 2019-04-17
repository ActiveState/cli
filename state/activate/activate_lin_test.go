// +build linux

package activate

func (suite *ActivateTestSuite) TestExecuteWithNamespaceWithLang() {
	suite.rMock.MockFullRuntime()
	pjfile := suite.testExecuteWithNamespace(true)
	suite.Require().NotEmpty(pjfile.Languages)
	suite.Equal("Python", pjfile.Languages[0].Name)
}
