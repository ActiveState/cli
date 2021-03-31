package artifact

import (
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/pkg/platform/runtime/testhelper"
)

func version(s string) *string {
	return &s
}

// TestArtifactsFromRecipe ensures that we are able to parse a recipe correctly
// This is probably good to do, as it is more complicated
func TestArtifactsFromRecipe(t *testing.T) {
	tests := []struct {
		Name       string
		recipeName string
		expected   ArtifactRecipeMap
	}{
		{
			"camel recipe",
			"camel",
			ArtifactRecipeMap{
				strfmt.UUID("bdd5642b-928c-5770-9e12-5816c9676960"): ArtifactRecipe{
					Name: "python", Namespace: "language", ArtifactID: strfmt.UUID("bdd5642b-928c-5770-9e12-5816c9676960"), Dependencies: nil, RequestedByOrder: true, Version: version("3.7.4")},
				strfmt.UUID("decfc04f-5770-5663-8d00-e029402e6917"): ArtifactRecipe{
					Name: "json2", Namespace: "language/python", ArtifactID: strfmt.UUID("decfc04f-5770-5663-8d00-e029402e6917"), Dependencies: []ArtifactID{"bdd5642b-928c-5770-9e12-5816c9676960"}, RequestedByOrder: true, Version: version("0.4.0")},
				strfmt.UUID("e6997088-7854-5498-8c57-afbe4343036a"): ArtifactRecipe{
					Name: "wheel", Namespace: "language/python", ArtifactID: strfmt.UUID("e6997088-7854-5498-8c57-afbe4343036a"), Dependencies: []ArtifactID{"bdd5642b-928c-5770-9e12-5816c9676960"}, RequestedByOrder: false, Version: version("0.35.1")},
			},
		},
		{
			"alternative recipe",
			"perl-alternative-base",
			ArtifactRecipeMap{
				strfmt.UUID("b30ab2e5-4074-572c-8146-da692b1c9e45"): ArtifactRecipe{
					Name: "perl", Namespace: "language", ArtifactID: strfmt.UUID("b30ab2e5-4074-572c-8146-da692b1c9e45"), Dependencies: nil, RequestedByOrder: true, Version: version("5.32.1")},
			},
		},
		{
			"alternative with bundles",
			"perl-alternative-one-bundle",
			ArtifactRecipeMap{
				strfmt.UUID("c894fa23-0416-556d-9ca5-fdf9375595bc"): ArtifactRecipe{
					Name: "Testing", Namespace: "bundles/perl", ArtifactID: strfmt.UUID("c894fa23-0416-556d-9ca5-fdf9375595bc"), Dependencies: []ArtifactID{"288aa0db-c0e4-55e7-8f67-fc2da409be70", "5ad88c8a-bc8f-50a0-9f61-74856cd28017", "30dc7965-0a69-5686-831a-e563fa73a98c", "8c2f830d-1b31-5448-a0a4-aa9d8fcacc4b"}, RequestedByOrder: true, Version: version("1.00")},
				strfmt.UUID("b30ab2e5-4074-572c-8146-da692b1c9e45"): ArtifactRecipe{
					Name: "perl", Namespace: "language", ArtifactID: strfmt.UUID("b30ab2e5-4074-572c-8146-da692b1c9e45"), Dependencies: nil, RequestedByOrder: true, Version: version("5.32.1")},
				strfmt.UUID("48951744-f839-5031-8cf4-6e82a4be2089"): ArtifactRecipe{
					Name: "Data-UUID", Namespace: "language/perl", ArtifactID: strfmt.UUID("48951744-f839-5031-8cf4-6e82a4be2089"), Dependencies: []ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("1.226")},
				strfmt.UUID("0029ae25-8497-5130-8268-1f0fe26ccc77"): ArtifactRecipe{
					Name: "Importer", Namespace: "language/perl", ArtifactID: strfmt.UUID("0029ae25-8497-5130-8268-1f0fe26ccc77"), Dependencies: []ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.025")},
				strfmt.UUID("6591f01d-939d-5080-bb1a-7816ff4d020b"): ArtifactRecipe{
					Name: "Long-Jump", Namespace: "language/perl", ArtifactID: strfmt.UUID("6591f01d-939d-5080-bb1a-7816ff4d020b"), Dependencies: []ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.000001")},
				strfmt.UUID("7c541a6a-4dfd-5135-8b98-2b44b5d1a816"): ArtifactRecipe{
					Name: "Module-Pluggable", Namespace: "language/perl", ArtifactID: strfmt.UUID("7c541a6a-4dfd-5135-8b98-2b44b5d1a816"), Dependencies: []ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("5.2")},
				strfmt.UUID("7f8a7197-b277-5621-a6f3-7f2ef32d871b"): ArtifactRecipe{
					Name: "Scope-Guard", Namespace: "language/perl", ArtifactID: strfmt.UUID("7f8a7197-b277-5621-a6f3-7f2ef32d871b"), Dependencies: []ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.21")},
				strfmt.UUID("29983a5b-49c4-5cf4-a2c5-2490647d6910"): ArtifactRecipe{
					Name: "Sub-Info", Namespace: "language/perl", ArtifactID: strfmt.UUID("29983a5b-49c4-5cf4-a2c5-2490647d6910"), Dependencies: []ArtifactID{"0029ae25-8497-5130-8268-1f0fe26ccc77", "b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.002")},
				strfmt.UUID("4d95557d-2200-5a56-a809-4ea3d3502b20"): ArtifactRecipe{
					Name: "Term-Table", Namespace: "language/perl", ArtifactID: strfmt.UUID("4d95557d-2200-5a56-a809-4ea3d3502b20"), Dependencies: []ArtifactID{"0029ae25-8497-5130-8268-1f0fe26ccc77", "b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.015")},
				strfmt.UUID("288aa0db-c0e4-55e7-8f67-fc2da409be70"): ArtifactRecipe{
					Name: "Test2-Harness", Namespace: "language/perl", ArtifactID: strfmt.UUID("288aa0db-c0e4-55e7-8f67-fc2da409be70"), Dependencies: []ArtifactID{
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
				strfmt.UUID("282e3768-e12a-51ed-831f-7cbc212ba8bd"): ArtifactRecipe{
					Name: "Test2-Plugin-MemUsage", Namespace: "language/perl", ArtifactID: strfmt.UUID("282e3768-e12a-51ed-831f-7cbc212ba8bd"), Dependencies: []ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.002003")},
				strfmt.UUID("5ad88c8a-bc8f-50a0-9f61-74856cd28017"): ArtifactRecipe{
					Name: "Test2-Plugin-NoWarnings", Namespace: "language/perl", ArtifactID: strfmt.UUID("5ad88c8a-bc8f-50a0-9f61-74856cd28017"), Dependencies: []ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.06")},
				strfmt.UUID("c3e652a7-676e-594f-b87f-93d19122f3f4"): ArtifactRecipe{
					Name: "Test2-Plugin-UUID", Namespace: "language/perl", ArtifactID: strfmt.UUID("c3e652a7-676e-594f-b87f-93d19122f3f4"), Dependencies: []ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45", "48951744-f839-5031-8cf4-6e82a4be2089"}, RequestedByOrder: false, Version: version("0.002001")},
				strfmt.UUID("30dc7965-0a69-5686-831a-e563fa73a98c"): ArtifactRecipe{
					Name: "Test2-Suite", Namespace: "language/perl", ArtifactID: strfmt.UUID("30dc7965-0a69-5686-831a-e563fa73a98c"), Dependencies: []ArtifactID{"7c541a6a-4dfd-5135-8b98-2b44b5d1a816", "7f8a7197-b277-5621-a6f3-7f2ef32d871b", "0029ae25-8497-5130-8268-1f0fe26ccc77", "4d95557d-2200-5a56-a809-4ea3d3502b20", "b30ab2e5-4074-572c-8146-da692b1c9e45", "29983a5b-49c4-5cf4-a2c5-2490647d6910"}, RequestedByOrder: false, Version: version("0.000127")},
				strfmt.UUID("c1e8c6c4-ea11-55a4-b415-97da2d32121e"): ArtifactRecipe{
					Name: "goto-file", Namespace: "language/perl", ArtifactID: strfmt.UUID("c1e8c6c4-ea11-55a4-b415-97da2d32121e"), Dependencies: []ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.005")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			recipe := testhelper.LoadRecipe(t, tt.recipeName)
			res := NewMapFromRecipe(recipe)
			assert.Equal(t, tt.expected, res)
		})
	}
}

func TestRequestedArtifactChanges(t *testing.T) {
	oldVersion := "2.97001"
	newVersion := "4.02"
	tests := []struct {
		Name            string
		baseRecipeName  string
		newRecipeName   string
		expectedChanges ArtifactChangeset
	}{
		{
			"no changes",
			"perl-alternative-base",
			"perl-alternative-base",
			ArtifactChangeset{},
		},
		{
			"one package added",
			"perl-alternative-base",
			"perl-alternative-one-package",
			ArtifactChangeset{Added: []strfmt.UUID{"bfe02625-c7d6-5604-ae04-2e5b4c9592a2"}},
		},
		{
			"one package updated",
			"perl-alternative-one-package",
			"perl-alternative-one-update",
			ArtifactChangeset{
				Updated: []ArtifactUpdate{{FromID: "bfe02625-c7d6-5604-ae04-2e5b4c9592a2", FromVersion: &oldVersion, ToID: "f56acc9c-dd02-5cf8-97f9-a5cd015f4c7b", ToVersion: &newVersion}},
			},
		},
		{
			"one package removed",
			"perl-alternative-one-update",
			"perl-alternative-one-removed",
			ArtifactChangeset{
				Removed: []strfmt.UUID{"f56acc9c-dd02-5cf8-97f9-a5cd015f4c7b"},
			},
		},
		{
			"added bundle",
			"perl-alternative-base",
			"perl-alternative-one-bundle",
			ArtifactChangeset{
				Added: []strfmt.UUID{"c894fa23-0416-556d-9ca5-fdf9375595bc"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			old := testhelper.LoadRecipe(t, tt.baseRecipeName)
			oldArts := NewMapFromRecipe(old)
			new := testhelper.LoadRecipe(t, tt.newRecipeName)
			newArts := NewMapFromRecipe(new)
			res := NewArtifactChangesetByIDMap(oldArts, newArts, true)

			assert.ElementsMatch(t, tt.expectedChanges.Added, res.Added, "mis-matched added ids")
			assert.ElementsMatch(t, tt.expectedChanges.Removed, res.Removed, "mis-matched removed ids")
			assert.ElementsMatch(t, tt.expectedChanges.Updated, res.Updated, "mis-matched updates")
		})
	}

	t.Run("starting empty", func(t *testing.T) {
		var oldArts ArtifactRecipeMap
		new := testhelper.LoadRecipe(t, "perl-alternative-base")
		newArts := NewMapFromRecipe(new)
		res := NewArtifactChangesetByIDMap(oldArts, newArts, true)

		assert.Equal(t, []strfmt.UUID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, res.Added, "mis-matched added ids")
		assert.Len(t, res.Removed, 0, "mis-matched removed ids")
		assert.Len(t, res.Updated, 0, "mis-matched updates")
	})
}

func TestResolvedArtifactChanges(t *testing.T) {
	oldVersion := "2.97001"
	newVersion := "4.02"
	tests := []struct {
		Name            string
		baseRecipeName  string
		newRecipeName   string
		expectedChanges ArtifactChangeset
	}{
		{
			"no changes",
			"perl-alternative-base",
			"perl-alternative-base",
			ArtifactChangeset{Added: []strfmt.UUID{}, Removed: []strfmt.UUID{}},
		},
		{
			"one package added",
			"perl-alternative-base",
			"perl-alternative-one-package",
			ArtifactChangeset{
				Added:   []strfmt.UUID{"41dbce7b-0d0f-597b-bb6f-411a4fb0b829", "bfe02625-c7d6-5604-ae04-2e5b4c9592a2", "d51871fd-d270-5423-82b9-78b567c53636", "c62e933c-7f68-5e94-8fcd-5f978e3825b4", "279d6621-2756-5f82-b1d4-1bd7a41dfc57"},
				Removed: []strfmt.UUID{}, Updated: []ArtifactUpdate{}},
		},
		{
			"one package updated",
			"perl-alternative-one-package",
			"perl-alternative-one-update",
			ArtifactChangeset{
				Added:   []strfmt.UUID{},
				Removed: []strfmt.UUID{"41dbce7b-0d0f-597b-bb6f-411a4fb0b829", "d51871fd-d270-5423-82b9-78b567c53636", "c62e933c-7f68-5e94-8fcd-5f978e3825b4", "279d6621-2756-5f82-b1d4-1bd7a41dfc57"},
				Updated: []ArtifactUpdate{{FromID: "bfe02625-c7d6-5604-ae04-2e5b4c9592a2", FromVersion: &oldVersion, ToID: "f56acc9c-dd02-5cf8-97f9-a5cd015f4c7b", ToVersion: &newVersion}},
			},
		},
		{
			"one package removed",
			"perl-alternative-one-update",
			"perl-alternative-one-removed",
			ArtifactChangeset{
				Removed: []strfmt.UUID{"f56acc9c-dd02-5cf8-97f9-a5cd015f4c7b"},
			},
		},
		{
			"added bundle",
			"perl-alternative-base",
			"perl-alternative-one-bundle",
			ArtifactChangeset{
				Added: []strfmt.UUID{"288aa0db-c0e4-55e7-8f67-fc2da409be70", "c1e8c6c4-ea11-55a4-b415-97da2d32121e", "48951744-f839-5031-8cf4-6e82a4be2089", "0029ae25-8497-5130-8268-1f0fe26ccc77", "7f8a7197-b277-5621-a6f3-7f2ef32d871b", "29983a5b-49c4-5cf4-a2c5-2490647d6910", "c3e652a7-676e-594f-b87f-93d19122f3f4", "5ad88c8a-bc8f-50a0-9f61-74856cd28017", "30dc7965-0a69-5686-831a-e563fa73a98c", "c894fa23-0416-556d-9ca5-fdf9375595bc", "6591f01d-939d-5080-bb1a-7816ff4d020b", "7c541a6a-4dfd-5135-8b98-2b44b5d1a816", "4d95557d-2200-5a56-a809-4ea3d3502b20", "282e3768-e12a-51ed-831f-7cbc212ba8bd"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			old := testhelper.LoadRecipe(t, tt.baseRecipeName)
			oldArts := NewMapFromRecipe(old)
			new := testhelper.LoadRecipe(t, tt.newRecipeName)
			newArts := NewMapFromRecipe(new)
			res := NewArtifactChangesetByIDMap(oldArts, newArts, false)

			assert.ElementsMatch(t, tt.expectedChanges.Added, res.Added, "mis-matched added ids")
			assert.ElementsMatch(t, tt.expectedChanges.Removed, res.Removed, "mis-matched removed ids")
			assert.ElementsMatch(t, tt.expectedChanges.Updated, res.Updated, "mis-matched updates")
		})
	}
}

func TestArtifactDownloads(t *testing.T) {
	tests := []struct {
		Name      string
		BuildName string
		IsCamel   bool
		Expected  []ArtifactDownload
	}{
		{
			"just-perl",
			"perl-alternative-base",
			false,
			[]ArtifactDownload{
				{"b30ab2e5-4074-572c-8146-da692b1c9e45", "s3://as-builds/production/language/perl/5.32.1/3/b30ab2e5-4074-572c-8146-da692b1c9e45/artifact.tar.gz", ""},
			},
		},
		{
			"with-one-package",
			"perl-alternative-one-update",
			false,
			[]ArtifactDownload{
				{"b30ab2e5-4074-572c-8146-da692b1c9e45", "s3://as-builds/production/language/perl/5.32.1/3/b30ab2e5-4074-572c-8146-da692b1c9e45/artifact.tar.gz", ""},
				{"f56acc9c-dd02-5cf8-97f9-a5cd015f4c7b", "s3://as-builds/production/language/perl/JSON/4.02/4/f56acc9c-dd02-5cf8-97f9-a5cd015f4c7b/artifact.tar.gz", ""},
			},
		},
		{
			"perl-camel",
			"perl",
			true,
			[]ArtifactDownload{
				{"e88f6f1f-74c9-512e-9c9b-8c921a80c6fb", "https://s3.amazonaws.com/camel-builds/ActivePerl/x86_64-linux-glibc-2.17/20200424T172842Z/ActivePerl-5.28.1.0000-x86_64-linux-glibc-2.17-2a0758c3.tar.gz", ""},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			build := testhelper.LoadBuildResponse(t, tt.BuildName)
			var downloads []ArtifactDownload
			var err error
			if tt.IsCamel {
				downloads, err = NewDownloadsFromCamelBuild(build)
			} else {
				downloads, err = NewDownloadsFromBuild(build)
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.Expected, downloads)
		})
	}
}

func TestRecursiveDependencies(t *testing.T) {
	artifacts := ArtifactRecipeMap{
		ArtifactID("1"): ArtifactRecipe{
			Dependencies: []ArtifactID{"2", "3"}},
		ArtifactID("2"): ArtifactRecipe{
			Dependencies: []ArtifactID{"4", "5"}},
		ArtifactID("3"): ArtifactRecipe{
			Dependencies: []ArtifactID{"4", "6"}},
		ArtifactID("4"): ArtifactRecipe{Dependencies: nil},
		ArtifactID("5"): ArtifactRecipe{Dependencies: nil},
		ArtifactID("6"): ArtifactRecipe{Dependencies: []ArtifactID{"7"}},
		ArtifactID("7"): ArtifactRecipe{Dependencies: nil},
	}

	tests := []struct {
		name     string
		artfID   ArtifactID
		expected []ArtifactID
	}{
		{name: "root artifact", artfID: "1", expected: []ArtifactID{"2", "3", "4", "5", "6", "7"}},
		{name: "invalid artifact", artfID: "1234", expected: nil},
		{name: "no recursion", artfID: "2", expected: []ArtifactID{"4", "5"}},
		{name: "partial recursion", artfID: "3", expected: []ArtifactID{"4", "6", "7"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := RecursiveDependenciesFor(tt.artfID, artifacts)
			assert.ElementsMatch(t, tt.expected, res)
		})
	}
}
