package model_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/pkg/platform/model"
)

type VCSTestSuite struct {
	suite.Suite
}

func (suite *VCSTestSuite) TestNamespaceMatch() {
	suite.True(model.NamespaceMatch("platform", model.NamespacePlatform))
	suite.False(model.NamespaceMatch(" platform ", model.NamespacePlatform))
	suite.False(model.NamespaceMatch("not-platform", model.NamespacePlatform))

	suite.True(model.NamespaceMatch("language", model.NamespaceLanguage))
	suite.False(model.NamespaceMatch(" language ", model.NamespaceLanguage))
	suite.False(model.NamespaceMatch("not-language", model.NamespaceLanguage))

	suite.True(model.NamespaceMatch("language/foo/package", model.NamespacePackage))
	suite.False(model.NamespaceMatch(" language/foo/package ", model.NamespacePackage))

	suite.True(model.NamespaceMatch("pre-platform-installer", model.NamespacePrePlatform))
	suite.False(model.NamespaceMatch(" pre-platform-installer ", model.NamespacePrePlatform))
}

func TestVCSTestSuite(t *testing.T) {
	suite.Run(t, new(VCSTestSuite))
}
