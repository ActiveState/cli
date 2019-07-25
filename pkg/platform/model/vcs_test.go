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
	suite.True(model.NamespaceMatch("platform", model.NamespacePlatformMatch))
	suite.False(model.NamespaceMatch(" platform ", model.NamespacePlatformMatch))
	suite.False(model.NamespaceMatch("not-platform", model.NamespacePlatformMatch))

	suite.True(model.NamespaceMatch("language", model.NamespaceLanguageMatch))
	suite.False(model.NamespaceMatch(" language ", model.NamespaceLanguageMatch))
	suite.False(model.NamespaceMatch("not-language", model.NamespaceLanguageMatch))

	suite.True(model.NamespaceMatch("language/foo/package", model.NamespacePackageMatch))
	suite.False(model.NamespaceMatch(" language/foo/package ", model.NamespacePackageMatch))

	suite.True(model.NamespaceMatch("pre-platform-installer", model.NamespacePrePlatformMatch))
	suite.False(model.NamespaceMatch(" pre-platform-installer ", model.NamespacePrePlatformMatch))
}

func TestVCSTestSuite(t *testing.T) {
	suite.Run(t, new(VCSTestSuite))
}
