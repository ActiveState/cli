package model_test

import (
	"testing"

	gqlModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
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

	suite.True(model.NamespaceMatch("language/foo", model.NamespacePackageMatch))
	suite.False(model.NamespaceMatch(" language/foo", model.NamespacePackageMatch))

	suite.True(model.NamespaceMatch("bundles/foo", model.NamespaceBundlesMatch))
	suite.False(model.NamespaceMatch(" bundles/foo", model.NamespaceBundlesMatch))

	suite.True(model.NamespaceMatch("pre-platform-installer", model.NamespacePrePlatformMatch))
	suite.False(model.NamespaceMatch(" pre-platform-installer ", model.NamespacePrePlatformMatch))
}

func (suite *VCSTestSuite) TestChangesetFromRequirements() {
	tests := []struct {
		op   model.Operation
		reqs []*gqlModel.Requirement
		want model.Changeset
	}{
		{
			model.OperationAdded,
			[]*gqlModel.Requirement{
				{
					Checkpoint: mono_models.Checkpoint{
						Namespace:         "a-name",
						Requirement:       "a-req",
						VersionConstraint: "a-vercon",
					},
				},
				{
					Checkpoint: mono_models.Checkpoint{
						Namespace:         "b-name",
						Requirement:       "b-req",
						VersionConstraint: "b-vercon",
					},
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
			[]*gqlModel.Requirement{
				{
					Checkpoint: mono_models.Checkpoint{
						Namespace:         "x-name",
						Requirement:       "x-req",
						VersionConstraint: "x-vercon",
					},
				},
				{
					Checkpoint: mono_models.Checkpoint{
						Namespace:         "y-name",
						Requirement:       "y-req",
						VersionConstraint: "y-vercon",
					},
				},
				{
					Checkpoint: mono_models.Checkpoint{
						Namespace:         "z-name",
						Requirement:       "z-req",
						VersionConstraint: "z-vercon",
					},
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

