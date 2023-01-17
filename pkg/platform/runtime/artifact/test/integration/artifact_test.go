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

// TestArtifactsFromRecipe ensures that we are able to parse a recipe correctly
// This is probably good to do, as it is more complicated
func TestArtifactsFromRecipe(t *testing.T) {
	tests := []struct {
		Name       string
		recipeName string
		expected   artifact.ArtifactRecipeMap
	}{
		{
			"camel recipe",
			"camel",
			artifact.ArtifactRecipeMap{
				strfmt.UUID("bdd5642b-928c-5770-9e12-5816c9676960"): artifact.ArtifactRecipe{
					Name: "python", Namespace: "language", ArtifactID: strfmt.UUID("bdd5642b-928c-5770-9e12-5816c9676960"), Dependencies: nil, RequestedByOrder: true, Version: version("3.7.4")},
				strfmt.UUID("decfc04f-5770-5663-8d00-e029402e6917"): artifact.ArtifactRecipe{
					Name: "json2", Namespace: "language/python", ArtifactID: strfmt.UUID("decfc04f-5770-5663-8d00-e029402e6917"), Dependencies: []artifact.ArtifactID{"bdd5642b-928c-5770-9e12-5816c9676960"}, RequestedByOrder: true, Version: version("0.4.0")},
				strfmt.UUID("e6997088-7854-5498-8c57-afbe4343036a"): artifact.ArtifactRecipe{
					Name: "wheel", Namespace: "language/python", ArtifactID: strfmt.UUID("e6997088-7854-5498-8c57-afbe4343036a"), Dependencies: []artifact.ArtifactID{"bdd5642b-928c-5770-9e12-5816c9676960"}, RequestedByOrder: false, Version: version("0.35.1")},
				strfmt.UUID("060cc2b8-01e4-5afe-8618-c44ccb25a592"): artifact.ArtifactRecipe{
					ArtifactID:       "060cc2b8-01e4-5afe-8618-c44ccb25a592",
					Name:             "tix",
					Namespace:        "shared",
					Version:          version("8.4.3.6"),
					RequestedByOrder: false,
				},
				strfmt.UUID("06e6c26f-7645-5971-a6ad-277497bdec0c"): artifact.ArtifactRecipe{
					ArtifactID:       "06e6c26f-7645-5971-a6ad-277497bdec0c",
					Name:             "tcl",
					Namespace:        "shared",
					Version:          version("8.6.8"),
					RequestedByOrder: false,
				},
				strfmt.UUID("1931b61b-5e8b-5bce-a5ff-5663a0b9b9c3"): artifact.ArtifactRecipe{
					ArtifactID:       "1931b61b-5e8b-5bce-a5ff-5663a0b9b9c3",
					Name:             "expat",
					Namespace:        "shared",
					Version:          version("2.2.9"),
					RequestedByOrder: false,
				},
				strfmt.UUID("2a2dc52f-8324-59bf-ae5e-082ea2468a28"): artifact.ArtifactRecipe{
					ArtifactID:       "2a2dc52f-8324-59bf-ae5e-082ea2468a28",
					Name:             "bsddb",
					Namespace:        "shared",
					Version:          version("4.4.20"),
					RequestedByOrder: false,
				},
				strfmt.UUID("2c8a61b6-995c-5e59-8d52-fac8604c3e88"): artifact.ArtifactRecipe{
					ArtifactID:       "2c8a61b6-995c-5e59-8d52-fac8604c3e88",
					Name:             "tk",
					Namespace:        "shared",
					Version:          version("8.6.8"),
					RequestedByOrder: false,
				},
				strfmt.UUID("7b923ae1-94a9-574c-bb9e-a89163f0ccb8"): artifact.ArtifactRecipe{
					ArtifactID:       "7b923ae1-94a9-574c-bb9e-a89163f0ccb8",
					Name:             "zlib",
					Namespace:        "shared",
					Version:          version("1.2.11"),
					RequestedByOrder: false,
				},
				strfmt.UUID("7b9a5527-a6d0-50b9-a6fe-0c5c73d242cd"): artifact.ArtifactRecipe{
					ArtifactID:       "7b9a5527-a6d0-50b9-a6fe-0c5c73d242cd",
					Name:             "sqlite3",
					Namespace:        "shared",
					Version:          version("3.15.2"),
					RequestedByOrder: false,
				},
				strfmt.UUID("f6f7099a-2a86-5a30-8098-a111661cfbc5"): artifact.ArtifactRecipe{
					ArtifactID:       "f6f7099a-2a86-5a30-8098-a111661cfbc5",
					Name:             "bzip2",
					Namespace:        "shared",
					Version:          version("1.0.6"),
					RequestedByOrder: false,
				},
				strfmt.UUID("fb916bb5-73f9-55f8-9a01-ad79a876e00c"): artifact.ArtifactRecipe{
					ArtifactID:       "fb916bb5-73f9-55f8-9a01-ad79a876e00c",
					Name:             "openssl",
					Namespace:        "shared",
					Version:          version("1.11.0.7"),
					RequestedByOrder: false,
				},
			},
		},
		{
			"alternative recipe",
			"perl-alternative-base",
			artifact.ArtifactRecipeMap{
				strfmt.UUID("b30ab2e5-4074-572c-8146-da692b1c9e45"): artifact.ArtifactRecipe{
					Name: "perl", Namespace: "language", ArtifactID: strfmt.UUID("b30ab2e5-4074-572c-8146-da692b1c9e45"), Dependencies: nil, RequestedByOrder: true, Version: version("5.32.1")},
			},
		},
		{
			"alternative with bundles",
			"perl-alternative-one-bundle",
			artifact.ArtifactRecipeMap{
				strfmt.UUID("c894fa23-0416-556d-9ca5-fdf9375595bc"): artifact.ArtifactRecipe{
					Name: "Testing", Namespace: "bundles/perl", ArtifactID: strfmt.UUID("c894fa23-0416-556d-9ca5-fdf9375595bc"), Dependencies: []artifact.ArtifactID{"288aa0db-c0e4-55e7-8f67-fc2da409be70", "5ad88c8a-bc8f-50a0-9f61-74856cd28017", "30dc7965-0a69-5686-831a-e563fa73a98c", "8c2f830d-1b31-5448-a0a4-aa9d8fcacc4b"}, RequestedByOrder: true, Version: version("1.00")},
				strfmt.UUID("b30ab2e5-4074-572c-8146-da692b1c9e45"): artifact.ArtifactRecipe{
					Name: "perl", Namespace: "language", ArtifactID: strfmt.UUID("b30ab2e5-4074-572c-8146-da692b1c9e45"), Dependencies: nil, RequestedByOrder: true, Version: version("5.32.1")},
				strfmt.UUID("48951744-f839-5031-8cf4-6e82a4be2089"): artifact.ArtifactRecipe{
					Name: "Data-UUID", Namespace: "language/perl", ArtifactID: strfmt.UUID("48951744-f839-5031-8cf4-6e82a4be2089"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("1.226")},
				strfmt.UUID("0029ae25-8497-5130-8268-1f0fe26ccc77"): artifact.ArtifactRecipe{
					Name: "Importer", Namespace: "language/perl", ArtifactID: strfmt.UUID("0029ae25-8497-5130-8268-1f0fe26ccc77"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.025")},
				strfmt.UUID("6591f01d-939d-5080-bb1a-7816ff4d020b"): artifact.ArtifactRecipe{
					Name: "Long-Jump", Namespace: "language/perl", ArtifactID: strfmt.UUID("6591f01d-939d-5080-bb1a-7816ff4d020b"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.000001")},
				strfmt.UUID("7c541a6a-4dfd-5135-8b98-2b44b5d1a816"): artifact.ArtifactRecipe{
					Name: "Module-Pluggable", Namespace: "language/perl", ArtifactID: strfmt.UUID("7c541a6a-4dfd-5135-8b98-2b44b5d1a816"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("5.2")},
				strfmt.UUID("7f8a7197-b277-5621-a6f3-7f2ef32d871b"): artifact.ArtifactRecipe{
					Name: "Scope-Guard", Namespace: "language/perl", ArtifactID: strfmt.UUID("7f8a7197-b277-5621-a6f3-7f2ef32d871b"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.21")},
				strfmt.UUID("29983a5b-49c4-5cf4-a2c5-2490647d6910"): artifact.ArtifactRecipe{
					Name: "Sub-Info", Namespace: "language/perl", ArtifactID: strfmt.UUID("29983a5b-49c4-5cf4-a2c5-2490647d6910"), Dependencies: []artifact.ArtifactID{"0029ae25-8497-5130-8268-1f0fe26ccc77", "b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.002")},
				strfmt.UUID("4d95557d-2200-5a56-a809-4ea3d3502b20"): artifact.ArtifactRecipe{
					Name: "Term-Table", Namespace: "language/perl", ArtifactID: strfmt.UUID("4d95557d-2200-5a56-a809-4ea3d3502b20"), Dependencies: []artifact.ArtifactID{"0029ae25-8497-5130-8268-1f0fe26ccc77", "b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.015")},
				strfmt.UUID("288aa0db-c0e4-55e7-8f67-fc2da409be70"): artifact.ArtifactRecipe{
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
				strfmt.UUID("282e3768-e12a-51ed-831f-7cbc212ba8bd"): artifact.ArtifactRecipe{
					Name: "Test2-Plugin-MemUsage", Namespace: "language/perl", ArtifactID: strfmt.UUID("282e3768-e12a-51ed-831f-7cbc212ba8bd"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.002003")},
				strfmt.UUID("5ad88c8a-bc8f-50a0-9f61-74856cd28017"): artifact.ArtifactRecipe{
					Name: "Test2-Plugin-NoWarnings", Namespace: "language/perl", ArtifactID: strfmt.UUID("5ad88c8a-bc8f-50a0-9f61-74856cd28017"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.06")},
				strfmt.UUID("c3e652a7-676e-594f-b87f-93d19122f3f4"): artifact.ArtifactRecipe{
					Name: "Test2-Plugin-UUID", Namespace: "language/perl", ArtifactID: strfmt.UUID("c3e652a7-676e-594f-b87f-93d19122f3f4"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45", "48951744-f839-5031-8cf4-6e82a4be2089"}, RequestedByOrder: false, Version: version("0.002001")},
				strfmt.UUID("30dc7965-0a69-5686-831a-e563fa73a98c"): artifact.ArtifactRecipe{
					Name: "Test2-Suite", Namespace: "language/perl", ArtifactID: strfmt.UUID("30dc7965-0a69-5686-831a-e563fa73a98c"), Dependencies: []artifact.ArtifactID{"7c541a6a-4dfd-5135-8b98-2b44b5d1a816", "7f8a7197-b277-5621-a6f3-7f2ef32d871b", "0029ae25-8497-5130-8268-1f0fe26ccc77", "4d95557d-2200-5a56-a809-4ea3d3502b20", "b30ab2e5-4074-572c-8146-da692b1c9e45", "29983a5b-49c4-5cf4-a2c5-2490647d6910"}, RequestedByOrder: false, Version: version("0.000127")},
				strfmt.UUID("c1e8c6c4-ea11-55a4-b415-97da2d32121e"): artifact.ArtifactRecipe{
					Name: "goto-file", Namespace: "language/perl", ArtifactID: strfmt.UUID("c1e8c6c4-ea11-55a4-b415-97da2d32121e"), Dependencies: []artifact.ArtifactID{"b30ab2e5-4074-572c-8146-da692b1c9e45"}, RequestedByOrder: false, Version: version("0.005")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			recipe := testhelper.LoadRecipe(t, tt.recipeName)
			res := artifact.NewMapFromRecipe(recipe)
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
		expectedChanges artifact.ArtifactChangeset
	}{
		{
			"no camel changes",
			"camel",
			"camel",
			artifact.ArtifactChangeset{},
		},
		{
			"no changes",
			"perl-alternative-base",
			"perl-alternative-base",
			artifact.ArtifactChangeset{},
		},
		{
			"one package added",
			"perl-alternative-base",
			"perl-alternative-one-package",
			artifact.ArtifactChangeset{Added: []strfmt.UUID{"bfe02625-c7d6-5604-ae04-2e5b4c9592a2"}},
		},
		{
			"one package updated",
			"perl-alternative-one-package",
			"perl-alternative-one-update",
			artifact.ArtifactChangeset{
				Updated: []artifact.ArtifactUpdate{{FromID: "bfe02625-c7d6-5604-ae04-2e5b4c9592a2", FromVersion: &oldVersion, ToID: "f56acc9c-dd02-5cf8-97f9-a5cd015f4c7b", ToVersion: &newVersion}},
			},
		},
		{
			"one package removed",
			"perl-alternative-one-update",
			"perl-alternative-one-removed",
			artifact.ArtifactChangeset{
				Removed: []strfmt.UUID{"f56acc9c-dd02-5cf8-97f9-a5cd015f4c7b"},
			},
		},
		{
			"added bundle",
			"perl-alternative-base",
			"perl-alternative-one-bundle",
			artifact.ArtifactChangeset{
				Added: []strfmt.UUID{"c894fa23-0416-556d-9ca5-fdf9375595bc"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			old := testhelper.LoadRecipe(t, tt.baseRecipeName)
			oldArts := artifact.NewMapFromRecipe(old)
			new := testhelper.LoadRecipe(t, tt.newRecipeName)
			newArts := artifact.NewMapFromRecipe(new)
			res := artifact.NewArtifactChangesetByIDMap(oldArts, newArts, true)

			assert.ElementsMatch(t, tt.expectedChanges.Added, res.Added, "mis-matched added ids")
			assert.ElementsMatch(t, tt.expectedChanges.Removed, res.Removed, "mis-matched removed ids")
			assert.ElementsMatch(t, tt.expectedChanges.Updated, res.Updated, "mis-matched updates")
		})
	}

	t.Run("starting empty", func(t *testing.T) {
		var oldArts artifact.ArtifactRecipeMap
		new := testhelper.LoadRecipe(t, "perl-alternative-base")
		newArts := artifact.NewMapFromRecipe(new)
		res := artifact.NewArtifactChangesetByIDMap(oldArts, newArts, true)

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
		expectedChanges artifact.ArtifactChangeset
	}{
		{
			"no changes",
			"perl-alternative-base",
			"perl-alternative-base",
			artifact.ArtifactChangeset{Added: []strfmt.UUID{}, Removed: []strfmt.UUID{}},
		},
		{
			"one package added",
			"perl-alternative-base",
			"perl-alternative-one-package",
			artifact.ArtifactChangeset{
				Added:   []strfmt.UUID{"41dbce7b-0d0f-597b-bb6f-411a4fb0b829", "bfe02625-c7d6-5604-ae04-2e5b4c9592a2", "d51871fd-d270-5423-82b9-78b567c53636", "c62e933c-7f68-5e94-8fcd-5f978e3825b4", "279d6621-2756-5f82-b1d4-1bd7a41dfc57"},
				Removed: []strfmt.UUID{}, Updated: []artifact.ArtifactUpdate{}},
		},
		{
			"one package updated",
			"perl-alternative-one-package",
			"perl-alternative-one-update",
			artifact.ArtifactChangeset{
				Added:   []strfmt.UUID{},
				Removed: []strfmt.UUID{"41dbce7b-0d0f-597b-bb6f-411a4fb0b829", "d51871fd-d270-5423-82b9-78b567c53636", "c62e933c-7f68-5e94-8fcd-5f978e3825b4", "279d6621-2756-5f82-b1d4-1bd7a41dfc57"},
				Updated: []artifact.ArtifactUpdate{{FromID: "bfe02625-c7d6-5604-ae04-2e5b4c9592a2", FromVersion: &oldVersion, ToID: "f56acc9c-dd02-5cf8-97f9-a5cd015f4c7b", ToVersion: &newVersion}},
			},
		},
		{
			"one package removed",
			"perl-alternative-one-update",
			"perl-alternative-one-removed",
			artifact.ArtifactChangeset{
				Removed: []strfmt.UUID{"f56acc9c-dd02-5cf8-97f9-a5cd015f4c7b"},
			},
		},
		{
			"added bundle",
			"perl-alternative-base",
			"perl-alternative-one-bundle",
			artifact.ArtifactChangeset{
				Added: []strfmt.UUID{"288aa0db-c0e4-55e7-8f67-fc2da409be70", "c1e8c6c4-ea11-55a4-b415-97da2d32121e", "48951744-f839-5031-8cf4-6e82a4be2089", "0029ae25-8497-5130-8268-1f0fe26ccc77", "7f8a7197-b277-5621-a6f3-7f2ef32d871b", "29983a5b-49c4-5cf4-a2c5-2490647d6910", "c3e652a7-676e-594f-b87f-93d19122f3f4", "5ad88c8a-bc8f-50a0-9f61-74856cd28017", "30dc7965-0a69-5686-831a-e563fa73a98c", "c894fa23-0416-556d-9ca5-fdf9375595bc", "6591f01d-939d-5080-bb1a-7816ff4d020b", "7c541a6a-4dfd-5135-8b98-2b44b5d1a816", "4d95557d-2200-5a56-a809-4ea3d3502b20", "282e3768-e12a-51ed-831f-7cbc212ba8bd"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			old := testhelper.LoadRecipe(t, tt.baseRecipeName)
			oldArts := artifact.NewMapFromRecipe(old)
			new := testhelper.LoadRecipe(t, tt.newRecipeName)
			newArts := artifact.NewMapFromRecipe(new)
			res := artifact.NewArtifactChangesetByIDMap(oldArts, newArts, false)

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
				downloads, _, err = artifact.NewDownloadsFromBuild(build)
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.Expected, downloads)
		})
	}
}

func TestRecursiveDependencies(t *testing.T) {
	artifacts := artifact.ArtifactRecipeMap{
		artifact.ArtifactID("1"): artifact.ArtifactRecipe{
			Dependencies: []artifact.ArtifactID{"2", "3"}},
		artifact.ArtifactID("2"): artifact.ArtifactRecipe{
			Dependencies: []artifact.ArtifactID{"4", "5"}},
		artifact.ArtifactID("3"): artifact.ArtifactRecipe{
			Dependencies: []artifact.ArtifactID{"4", "6"}},
		artifact.ArtifactID("4"): artifact.ArtifactRecipe{Dependencies: nil},
		artifact.ArtifactID("5"): artifact.ArtifactRecipe{Dependencies: nil},
		artifact.ArtifactID("6"): artifact.ArtifactRecipe{Dependencies: []artifact.ArtifactID{"7"}},
		artifact.ArtifactID("7"): artifact.ArtifactRecipe{Dependencies: nil},
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
