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

	suite.True(model.NamespaceMatch("language/foo", model.NamespaceLanguagePackageMatch))
	suite.False(model.NamespaceMatch(" language/foo", model.NamespaceLanguagePackageMatch))

	suite.True(model.NamespaceMatch("bundles/foo", model.NamespaceBundlesPackageMatch))
	suite.False(model.NamespaceMatch(" bundles/foo", model.NamespaceBundlesPackageMatch))

	suite.True(model.NamespaceMatch("pre-platform-installer", model.NamespacePrePlatformMatch))
	suite.False(model.NamespaceMatch(" pre-platform-installer ", model.NamespacePrePlatformMatch))
}

func (suite *VCSTestSuite) TestChangesetFromRequirements() {
	tests := []struct {
		op   model.Operation
		reqs model.Checkpoint
		want model.Changeset
	}{
		{
			model.OperationAdded,
			model.Checkpoint{
				{
					Namespace:         "a-name",
					Requirement:       "a-req",
					VersionConstraint: "a-vercon",
				},
				{
					Namespace:         "b-name",
					Requirement:       "b-req",
					VersionConstraint: "b-vercon",
				},
			},
			model.Changeset{
				{
					Operation:         string(model.OperationAdded),
					Namespace:         "a-name",
					Requirement:       "a-req",
					VersionConstraint: "a-vercon",
				},
				{
					Operation:         string(model.OperationAdded),
					Namespace:         "b-name",
					Requirement:       "b-req",
					VersionConstraint: "b-vercon",
				},
			},
		},
		{
			model.OperationRemoved,
			model.Checkpoint{
				{
					Namespace:         "x-name",
					Requirement:       "x-req",
					VersionConstraint: "x-vercon",
				},
				{
					Namespace:         "y-name",
					Requirement:       "y-req",
					VersionConstraint: "y-vercon",
				},
				{
					Namespace:         "z-name",
					Requirement:       "z-req",
					VersionConstraint: "z-vercon",
				},
			},
			model.Changeset{
				{
					Operation:         string(model.OperationRemoved),
					Namespace:         "x-name",
					Requirement:       "x-req",
					VersionConstraint: "x-vercon",
				},
				{
					Operation:         string(model.OperationRemoved),
					Namespace:         "y-name",
					Requirement:       "y-req",
					VersionConstraint: "y-vercon",
				},
				{
					Operation:         string(model.OperationRemoved),
					Namespace:         "z-name",
					Requirement:       "z-req",
					VersionConstraint: "z-vercon",
				},
			},
		},
	}

	for _, tt := range tests {
		got := model.ChangesetFromRequirements(tt.op, tt.reqs)
		suite.Equal(tt.want, got)
	}
}

func TestVCSTestSuite(t *testing.T) {
	suite.Run(t, new(VCSTestSuite))
}
