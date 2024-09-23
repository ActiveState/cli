package model

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/suite"
	gqlModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

type VCSTestSuite struct {
	suite.Suite
}

func (suite *VCSTestSuite) TestNamespaceMatch() {
	suite.True(NamespaceMatch("platform", NamespacePlatformMatch))
	suite.False(NamespaceMatch(" platform ", NamespacePlatformMatch))
	suite.False(NamespaceMatch("not-platform", NamespacePlatformMatch))

	suite.True(NamespaceMatch("language", NamespaceLanguageMatch))
	suite.False(NamespaceMatch(" language ", NamespaceLanguageMatch))
	suite.False(NamespaceMatch("not-language", NamespaceLanguageMatch))

	suite.True(NamespaceMatch("language/foo", NamespacePackageMatch))
	suite.False(NamespaceMatch(" language/foo", NamespacePackageMatch))

	suite.True(NamespaceMatch("bundles/foo", NamespaceBundlesMatch))
	suite.False(NamespaceMatch(" bundles/foo", NamespaceBundlesMatch))

	suite.True(NamespaceMatch("pre-platform-installer", NamespacePrePlatformMatch))
	suite.False(NamespaceMatch(" pre-platform-installer ", NamespacePrePlatformMatch))
}

func (suite *VCSTestSuite) TestChangesetFromRequirements() {
	tests := []struct {
		op   Operation
		reqs []*gqlModel.Requirement
		want Changeset
	}{
		{
			OperationAdded,
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
			Changeset{
				{
					Operation:         string(OperationAdded),
					Namespace:         "a-name",
					Requirement:       "a-req",
					VersionConstraint: "a-vercon",
				},
				{
					Operation:         string(OperationAdded),
					Namespace:         "b-name",
					Requirement:       "b-req",
					VersionConstraint: "b-vercon",
				},
			},
		},
		{
			OperationRemoved,
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
			Changeset{
				{
					Operation:         string(OperationRemoved),
					Namespace:         "x-name",
					Requirement:       "x-req",
					VersionConstraint: "x-vercon",
				},
				{
					Operation:         string(OperationRemoved),
					Namespace:         "y-name",
					Requirement:       "y-req",
					VersionConstraint: "y-vercon",
				},
				{
					Operation:         string(OperationRemoved),
					Namespace:         "z-name",
					Requirement:       "z-req",
					VersionConstraint: "z-vercon",
				},
			},
		},
	}

	for _, tt := range tests {
		got := ChangesetFromRequirements(tt.op, tt.reqs)
		suite.Equal(tt.want, got)
	}
}

func (suite *VCSTestSuite) TestVersionStringToConstraints() {
	tests := []struct {
		version string
		want    []*mono_models.Constraint
	}{
		{
			"3.10.10",
			[]*mono_models.Constraint{
				{Comparator: "eq", Version: "3.10.10"},
			},
		},
		{
			"3.10.x",
			[]*mono_models.Constraint{
				{Comparator: "gte", Version: "3.10"},
				{Comparator: "lt", Version: "3.11"},
			},
		},
		{
			"2.x",
			[]*mono_models.Constraint{
				{Comparator: "gte", Version: "2"},
				{Comparator: "lt", Version: "3"},
			},
		},
	}

	for _, tt := range tests {
		got, err := versionStringToConstraints(tt.version)
		suite.NoError(err)
		suite.Equal(tt.want, got)
	}
}

func TestVCSTestSuite(t *testing.T) {
	suite.Run(t, new(VCSTestSuite))
}

func TestParseNamespace(t *testing.T) {
	tests := []struct {
		ns   string
		want NamespaceType
	}{
		{
			"language/python",
			NamespacePackage,
		},
		{
			"bundles/python",
			NamespaceBundle,
		},
		{
			"language",
			NamespaceLanguage,
		},
		{
			"platform",
			NamespacePlatform,
		},
		{
			"private/org",
			NamespaceOrg,
		},
		{
			"raw/foo/bar",
			NamespaceRaw,
		},
	}
	for _, tt := range tests {
		t.Run(tt.ns, func(t *testing.T) {
			if got := ParseNamespace(tt.ns); got.Type().name != tt.want.name {
				t.Errorf("ParseNamespace() = %v, want %v", got.Type().name, tt.want.name)
			}
		})
	}
}
