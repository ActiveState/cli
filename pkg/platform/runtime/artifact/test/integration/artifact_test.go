package integration

import (
	"testing"

	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/pkg/platform/runtime/testhelper"
)

func version(s string) *string {
	return &s
}

// TestArtifactsFromBuildPlan ensures that we are able to parse a recipe correctly
// This is probably good to do, as it is more complicated
func TestArtifactsFromBuildPlan(t *testing.T) {
	tests := []struct {
		Name       string
		recipeName string
		expected   artifact.ArtifactBuildPlanMap
	}{
		{
			"initial build plan",
			"flask",
			artifact.ArtifactBuildPlanMap{
				strfmt.UUID("0f5a2e25-2879-5a3a-b1dc-ba6b89ee446f:"): artifact.ArtifactBuildPlan{
					Name: "python", Namespace: "language", ArtifactID: strfmt.UUID("0f5a2e25-2879-5a3a-b1dc-ba6b89ee446f"), RequestedByOrder: true, Version: version("3.10.2")},
				strfmt.UUID("b30ab2e5-4074-572c-8146-da692b1c9e45"): artifact.ArtifactBuildPlan{
					Name: "perl", Namespace: "language", ArtifactID: strfmt.UUID("b30ab2e5-4074-572c-8146-da692b1c9e45"), Dependencies: nil, RequestedByOrder: true, Version: version("3.10.2")},
				strfmt.UUID("48951744-f839-5031-8cf4-6e82a4be2089"): artifact.ArtifactBuildPlan{
					Name: "Data-UUID", Namespace: "language/perl", ArtifactID: strfmt.UUID("48951744-f839-5031-8cf4-6e82a4be2089"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("1.226")},
				strfmt.UUID("0029ae25-8497-5130-8268-1f0fe26ccc77"): artifact.ArtifactBuildPlan{
					Name: "Importer", Namespace: "language/perl", ArtifactID: strfmt.UUID("0029ae25-8497-5130-8268-1f0fe26ccc77"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.025")},
				strfmt.UUID("6591f01d-939d-5080-bb1a-7816ff4d020b"): artifact.ArtifactBuildPlan{
					Name: "Long-Jump", Namespace: "language/perl", ArtifactID: strfmt.UUID("6591f01d-939d-5080-bb1a-7816ff4d020b"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.000001")},
				strfmt.UUID("7c541a6a-4dfd-5135-8b98-2b44b5d1a816"): artifact.ArtifactBuildPlan{
					Name: "Module-Pluggable", Namespace: "language/perl", ArtifactID: strfmt.UUID("7c541a6a-4dfd-5135-8b98-2b44b5d1a816"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("5.2")},
				strfmt.UUID("7f8a7197-b277-5621-a6f3-7f2ef32d871b"): artifact.ArtifactBuildPlan{
					Name: "Scope-Guard", Namespace: "language/perl", ArtifactID: strfmt.UUID("7f8a7197-b277-5621-a6f3-7f2ef32d871b"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.21")},
				strfmt.UUID("29983a5b-49c4-5cf4-a2c5-2490647d6910"): artifact.ArtifactBuildPlan{
					Name: "Sub-Info", Namespace: "language/perl", ArtifactID: strfmt.UUID("29983a5b-49c4-5cf4-a2c5-2490647d6910"), Dependencies: []artifact.ArtifactID{"0029ae25-8497-5130-8268-1f0fe26ccc77", "b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.002")},
				strfmt.UUID("4d95557d-2200-5a56-a809-4ea3d3502b20"): artifact.ArtifactBuildPlan{
					Name: "Term-Table", Namespace: "language/perl", ArtifactID: strfmt.UUID("4d95557d-2200-5a56-a809-4ea3d3502b20"), Dependencies: []artifact.ArtifactID{"0029ae25-8497-5130-8268-1f0fe26ccc77", "b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.015")},
				strfmt.UUID("288aa0db-c0e4-55e7-8f67-fc2da409be70"): artifact.ArtifactBuildPlan{
					Name: "Test2-Harness", Namespace: "language/perl", ArtifactID: strfmt.UUID("288aa0db-c0e4-55e7-8f67-fc2da409be70"), Dependencies: []artifact.ArtifactID{
						"7f8a7197-b277-5621-a6f3-7f2ef32d871b",
						"0029ae25-8497-5130-8268-1f0fe26ccc77",
						"c3e652a7-676e-594f-b87f-93d19122f3f4",
						"6591f01d-939d-5080-bb1a-7816ff4d020b",
						"4d95557d-2200-5a56-a809-4ea3d3502b20",
						"b30ab2e5-4074-572c-8146-da692b1c9e45",
						"282e3768-e12a-51ed-831f-7cbc212ba8bd",
						"48951744-f839-5031-8cf4-6e82a4be2089",
						"c1e8c6c4-ea11-55a4-b415-97da2d32121e",
						"30dc7965-0a69-5686-831a-e563fa73a98c",
					}, RequestedByOrder: false, Version: version("1.000042")},
				strfmt.UUID("282e3768-e12a-51ed-831f-7cbc212ba8bd"): artifact.ArtifactBuildPlan{
					Name: "Test2-Plugin-MemUsage", Namespace: "language/perl", ArtifactID: strfmt.UUID("282e3768-e12a-51ed-831f-7cbc212ba8bd"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.002003")},
				strfmt.UUID("5ad88c8a-bc8f-50a0-9f61-74856cd28017"): artifact.ArtifactBuildPlan{
					Name: "Test2-Plugin-NoWarnings", Namespace: "language/perl", ArtifactID: strfmt.UUID("5ad88c8a-bc8f-50a0-9f61-74856cd28017"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.06")},
				strfmt.UUID("c3e652a7-676e-594f-b87f-93d19122f3f4"): artifact.ArtifactBuildPlan{
					Name: "Test2-Plugin-UUID", Namespace: "language/perl", ArtifactID: strfmt.UUID("c3e652a7-676e-594f-b87f-93d19122f3f4"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45", "48951744-f839-5031-8cf4-6e82a4be2089"}, RequestedByOrder: false, Version: version("0.002001")},
				strfmt.UUID("30dc7965-0a69-5686-831a-e563fa73a98c"): artifact.ArtifactBuildPlan{
					Name: "Test2-Suite", Namespace: "language/perl", ArtifactID: strfmt.UUID("30dc7965-0a69-5686-831a-e563fa73a98c"), Dependencies: []artifact.ArtifactID{"7c541a6a-4dfd-5135-8b98-2b44b5d1a816", "7f8a7197-b277-5621-a6f3-7f2ef32d871b", "0029ae25-8497-5130-8268-1f0fe26ccc77", "4d95557d-2200-5a56-a809-4ea3d3502b20", "b30ab2e5-4074-572c-8146-da692b1c9e45", "29983a5b-49c4-5cf4-a2c5-2490647d6910"}, RequestedByOrder: false, Version: version("0.000127")},
				strfmt.UUID("c1e8c6c4-ea11-55a4-b415-97da2d32121e"): artifact.ArtifactBuildPlan{
					Name: "goto-file", Namespace: "language/perl", ArtifactID: strfmt.UUID("c1e8c6c4-ea11-55a4-b415-97da2d32121e"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.005")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			bp := testhelper.LoadBuildPlan(t, tt.recipeName)
			res := artifact.NewMapFromBuildPlan(bp.Project.Commit.Build)
			// fmt.Printf("Result: %+v\n", res)
			assert.Equal(t, tt.expected, res)
		})
	}
}

func TestArtifactDownloads(t *testing.T) {
	tests := []struct {
		Name      string
		BuildName string
		IsCamel   bool
		Expected  []artifact.ArtifactDownload
	}{
		{
			"just-perl",
			"perl-alternative-base",
			false,
			[]artifact.ArtifactDownload{
				{ArtifactID: "b30ab2e5-4074-572c-8146-da692b1c9e45", UnsignedURI: "s3://as-builds/production/language/perl/5.32.1/3/b30ab2e5-4074-572c-8146-da692b1c9e45/artifact.tar.gz"},
			},
		},
		{
			"with-one-package",
			"perl-alternative-one-update",
			false,
			[]artifact.ArtifactDownload{
				{ArtifactID: "b30ab2e5-4074-572c-8146-da692b1c9e45", UnsignedURI: "s3://as-builds/production/language/perl/5.32.1/3/b30ab2e5-4074-572c-8146-da692b1c9e45/artifact.tar.gz"},
				{ArtifactID: "f56acc9c-dd02-5cf8-97f9-a5cd015f4c7b", UnsignedURI: "s3://as-builds/production/language/perl/JSON/4.02/4/f56acc9c-dd02-5cf8-97f9-a5cd015f4c7b/artifact.tar.gz"},
			},
		},
		{
			"perl-camel",
			"perl",
			true,
			[]artifact.ArtifactDownload{
				{ArtifactID: "e88f6f1f-74c9-512e-9c9b-8c921a80c6fb", UnsignedURI: "https://s3.amazonaws.com/camel-builds/ActivePerl/x86_64-linux-glibc-2.17/20200424T172842Z/ActivePerl-5.28.1.0000-x86_64-linux-glibc-2.17-2a0758c3.tar.gz"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			build := testhelper.LoadBuildResponse(t, tt.BuildName)
			var downloads []artifact.ArtifactDownload
			var err error
			if tt.IsCamel {
				downloads, err = artifact.NewDownloadsFromCamelBuild(build)
			} else {
				downloads, err = artifact.NewDownloadsFromBuild(build)
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.Expected, downloads)
		})
	}
}

func TestRecursiveDependencies(t *testing.T) {
	artifacts := artifact.ArtifactBuildPlanMap{
		artifact.ArtifactID("1"): artifact.ArtifactBuildPlan{
			Dependencies: []artifact.ArtifactID{"2", "3"}},
		artifact.ArtifactID("2"): artifact.ArtifactBuildPlan{
			Dependencies: []artifact.ArtifactID{"4", "5"}},
		artifact.ArtifactID("3"): artifact.ArtifactBuildPlan{
			Dependencies: []artifact.ArtifactID{"4", "6"}},
		artifact.ArtifactID("4"): artifact.ArtifactBuildPlan{Dependencies: nil},
		artifact.ArtifactID("5"): artifact.ArtifactBuildPlan{Dependencies: nil},
		artifact.ArtifactID("6"): artifact.ArtifactBuildPlan{Dependencies: []artifact.ArtifactID{"7"}},
		artifact.ArtifactID("7"): artifact.ArtifactBuildPlan{Dependencies: nil},
	}

	tests := []struct {
		name     string
		artfID   artifact.ArtifactID
		expected []artifact.ArtifactID
	}{
		{name: "root artifact", artfID: "1", expected: []artifact.ArtifactID{"2", "3", "4", "5", "6", "7"}},
		{name: "invalid artifact", artfID: "1234", expected: nil},
		{name: "no recursion", artfID: "2", expected: []artifact.ArtifactID{"4", "5"}},
		{name: "partial recursion", artfID: "3", expected: []artifact.ArtifactID{"4", "6", "7"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := artifact.RecursiveDependenciesFor(tt.artfID, artifacts)
			assert.ElementsMatch(t, tt.expected, res)
		})
	}
}
