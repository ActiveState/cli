package artifact

import (
	"sort"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/pkg/platform/runtime2/testhelper"
)

// TestArtifactsFromRecipe ensures that we are able to parse a recipe correctly
// This is probably good to do, as it is more complicated
func TestArtifactsFromRecipe(t *testing.T) {
	tests := []struct {
		Name                  string
		recipeName            string
		expectedArtifactNames []string // TODO: expect full artifact structure maybe
	}{
		{
			"camel recipe",
			"camel",
			[]string{"python", "json2", "wheel"},
		},
		{
			"alternative recipe",
			"perl-alternative-base",
			[]string{"perl"},
		},
		{
			"alternative with bundles",
			"perl-alternative-one-bundle",
			[]string{"Testing", "perl", "Data-UUID", "Importer", "Long-Jump", "Module-Pluggable", "Scope-Guard", "Sub-Info", "Term-Table", "Test2-Harness", "Test2-Plugin-MemUsage", "Test2-Plugin-NoWarnings", "Test2-Plugin-UUID", "Test2-Suite", "goto-file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			recipe := testhelper.LoadRecipe(t, tt.recipeName)
			res := NewMapFromRecipe(recipe)
			artSlice := funk.Map(res, func(_ ArtifactID, a ArtifactRecipe) ArtifactRecipe { return a }).([]ArtifactRecipe)
			sort.Slice(artSlice, func(i, j int) bool { return artSlice[i].RecipePosition < artSlice[j].RecipePosition })
			assert.Equal(t, tt.expectedArtifactNames, funk.Map(artSlice, func(a ArtifactRecipe) string { return a.Name }))
			// TODO add more assertions especially about dependencies
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
		Expected  []ArtifactDownload
	}{
		{
			"just-perl",
			"perl-alternative-base",
			[]ArtifactDownload{
				{"b30ab2e5-4074-572c-8146-da692b1c9e45", "s3://as-builds/production/language/perl/5.32.1/3/b30ab2e5-4074-572c-8146-da692b1c9e45/artifact.tar.gz", ""},
			},
		},
		{
			"with-one-package",
			"perl-alternative-one-update",
			[]ArtifactDownload{
				{"b30ab2e5-4074-572c-8146-da692b1c9e45", "s3://as-builds/production/language/perl/5.32.1/3/b30ab2e5-4074-572c-8146-da692b1c9e45/artifact.tar.gz", ""},
				{"f56acc9c-dd02-5cf8-97f9-a5cd015f4c7b", "s3://as-builds/production/language/perl/JSON/4.02/4/f56acc9c-dd02-5cf8-97f9-a5cd015f4c7b/artifact.tar.gz", ""},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			build := testhelper.LoadBuildResponse(t, tt.BuildName)
			downloads, err := NewDownloadsFromBuild(build)
			assert.NoError(t, err)
			assert.Equal(t, tt.Expected, downloads)
		})
	}
}
