// +build linux

package activate

func (suite *ActivateTestSuite) TestExecuteWithNamespaceWithLang() {
	suite.rMock.MockFullRuntime()
	pjfile := suite.testExecuteWithNamespace(true)
	suite.Require().Empty(pjfile.Languages)
}
