// +build linux

package activate

func (suite *ActivateTestSuite) TestExecuteWithNamespaceWithLang() {
	pjfile := suite.testExecuteWithNamespace()
	suite.Require().NotEmpty(pjfile.Languages)
	suite.Equal("Python", pjfile.Languages[0].Name)
}
