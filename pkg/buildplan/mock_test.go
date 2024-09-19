package buildplan

import (
	"github.com/ActiveState/cli/pkg/buildplan/raw"
)

// createMockArtifactWithCycles creates a mock artifact with a cycle.
// Unfortunately go doesn't support circular variable initialization, so we have to do it manually.
// Rather than have a highly nested structure, we'll create the artifacts and ingredients separately
// and then link them together in a function.
//
// The artifact cycle is:
// 00000000-0000-0000-0000-000000000001
//
//	-> 00000000-0000-0000-0000-000000000002
//	  -> 00000000-0000-0000-0000-000000000003
//	    -> 00000000-0000-0000-0000-000000000001 (Cycle back to the first artifact)
func createMockArtifactWithCycles() *Artifact {
	// Create the artifacts with placeholders
	artifact0001 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000001"}
	artifact0002 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000002"}
	artifact0003 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000003"}

	// Create the deepest ingredients and artifacts first
	artifact0003.children = []ArtifactRelation{
		{
			Artifact: artifact0001, // This creates an artifact cycle back to artifact0001
			Relation: RuntimeRelation,
		},
	}

	artifact0002.children = []ArtifactRelation{
		{
			Artifact: artifact0003,
			Relation: RuntimeRelation,
		},
	}

	artifact0001.children = []ArtifactRelation{
		{
			Artifact: artifact0002,
			Relation: RuntimeRelation,
		},
	}

	return artifact0001
}

// createMockArtifactWithRuntimeDeps creates a mock artifact with runtime dependencies.
// The dependencies are:
// 00000000-0000-0000-0000-000000000001
//
//	-> 00000000-0000-0000-0000-000000000002 (child)
//	  -> 00000000-0000-0000-0000-000000000003 (child)
func createMockArtifactWithRuntimeDeps() *Artifact {
	artifact0001 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000001"}
	artifact0002 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000002"}
	artifact0003 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000003"}
	artifact0004 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000004"}

	artifact0001.children = []ArtifactRelation{
		{
			Artifact: artifact0002,
			Relation: RuntimeRelation,
		},
	}

	artifact0002.children = []ArtifactRelation{
		{
			Artifact: artifact0003,
			Relation: RuntimeRelation,
		},
	}

	artifact0003.children = []ArtifactRelation{
		{
			Artifact: artifact0004,
			Relation: RuntimeRelation,
		},
	}

	return artifact0001
}

// createMockArtifactWithBuildTimeDeps creates a mock artifact with build time dependencies.
// The dependencies are:
// 00000000-0000-0000-0000-000000000001
//
//	-> 00000000-0000-0000-0000-000000000002 (child)
//	  -> 00000000-0000-0000-0000-000000000003 (child)
func createMockArtifactWithBuildTimeDeps() *Artifact {
	artifact0001 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000001"}
	artifact0002 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000002"}
	artifact0003 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000003"}

	artifact0001.children = []ArtifactRelation{
		{
			Artifact: artifact0002,
			Relation: BuildtimeRelation,
		},
	}

	artifact0002.children = []ArtifactRelation{
		{
			Artifact: artifact0003,
			Relation: BuildtimeRelation,
		},
	}

	return artifact0001
}

// createIngredientWithRuntimeDeps creates a mock ingredient with runtime dependencies.
// The dependencies are:
// 00000000-0000-0000-0000-000000000010 (Ingredient0010)
//
//	-> 00000000-0000-0000-0000-000000000001 (Artifact0001)
//	  -> 00000000-0000-0000-0000-000000000002 (Artifact child of Artifact0001)
//	    -> 00000000-0000-0000-0000-000000000020 (Ingredient0020)
//	      -> 00000000-0000-0000-0000-000000000003 (Artifact0003)
//	        -> 00000000-0000-0000-0000-000000000004 (Artifact child of Artifact0003)
//	          -> 00000000-0000-0000-0000-000000000030 (Ingredient0030)
func createIngredientWithRuntimeDeps() *Ingredient {
	artifact0001 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000001"}
	artifact0002 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000002"}
	artifact0003 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000003"}
	artifact0004 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000004"}

	ingredient0010 := &Ingredient{
		IngredientSource: &raw.IngredientSource{
			IngredientID: "00000000-0000-0000-0000-000000000010",
		},
		Artifacts: []*Artifact{
			artifact0001,
		},
	}

	ingredient0020 := &Ingredient{
		IngredientSource: &raw.IngredientSource{
			IngredientID: "00000000-0000-0000-0000-000000000020",
		},
		Artifacts: []*Artifact{
			artifact0002,
		},
	}

	ingredient0030 := &Ingredient{
		IngredientSource: &raw.IngredientSource{
			IngredientID: "00000000-0000-0000-0000-000000000030",
		},
		Artifacts: []*Artifact{
			artifact0003,
		},
	}

	artifact0001.children = []ArtifactRelation{
		{
			Artifact: artifact0002,
			Relation: RuntimeRelation,
		},
	}

	artifact0002.children = []ArtifactRelation{
		{
			Artifact: artifact0003,
			Relation: RuntimeRelation,
		},
	}
	artifact0002.Ingredients = []*Ingredient{ingredient0020}

	artifact0003.children = []ArtifactRelation{
		{
			Artifact: artifact0004,
			Relation: RuntimeRelation,
		},
	}
	artifact0003.Ingredients = []*Ingredient{ingredient0030}

	return ingredient0010
}

// createMockIngredientWithCycles creates a mock ingredient with a cycle and avoids
// the circular variable initialization problem and the need for a highly nested structure.
//
// Ingredient are a bit more complex than artifacts as we first traverse the artifacts related
// to the ingredient then the children of that artifact and finally the ingredients of those children.
//
// The ingredient cycle is:
// 00000000-0000-0000-0000-000000000010 (Ingredient0010)
//
//	-> 00000000-0000-0000-0000-000000000001 (Artifact0001)
//	  -> 00000000-0000-0000-0000-000000000002 (Child of Artifact0001)
//	    -> 00000000-0000-0000-0000-000000000020 (Ingredient0020)
//	      -> 00000000-0000-0000-0000-000000000003 (Artifact0003)
//	        -> 00000000-0000-0000-0000-000000000004 (Child of Artifact0003)
//	          -> 00000000-0000-0000-0000-000000000030 (Ingredient0030)
//	            -> 00000000-0000-0000-0000-000000000005 (Artifact0005)
//	              -> 00000000-0000-0000-0000-000000000006 (Child of Artifact0005)
//	                -> 00000000-0000-0000-0000-000000000010 (Ingredient0010 cycle back to the first ingredient)
func createMockIngredientWithCycles() *Ingredient {
	artifact0001 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000001"}
	artifact0002 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000002"}
	artifact0003 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000003"}
	artifact0004 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000004"}
	artifact0005 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000005"}
	artifact0006 := &Artifact{ArtifactID: "00000000-0000-0000-0000-000000000006"}

	ingredient0010 := &Ingredient{
		IngredientSource: &raw.IngredientSource{
			IngredientID: "00000000-0000-0000-0000-000000000010",
		},
		Artifacts: []*Artifact{
			artifact0001,
		},
	}

	ingredient0020 := &Ingredient{
		IngredientSource: &raw.IngredientSource{
			IngredientID: "00000000-0000-0000-0000-000000000020",
		},
		Artifacts: []*Artifact{
			artifact0003,
		},
	}

	ingredient0030 := &Ingredient{
		IngredientSource: &raw.IngredientSource{
			IngredientID: "00000000-0000-0000-0000-000000000030",
		},
		Artifacts: []*Artifact{
			artifact0005,
		},
	}

	artifact0001.children = []ArtifactRelation{
		{
			Artifact: artifact0002,
		},
	}

	artifact0003.children = []ArtifactRelation{
		{
			Artifact: artifact0004,
		},
	}

	artifact0005.children = []ArtifactRelation{
		{
			Artifact: artifact0006,
		},
	}

	artifact0002.Ingredients = []*Ingredient{
		ingredient0020,
	}

	artifact0004.Ingredients = []*Ingredient{
		ingredient0030,
	}

	artifact0006.Ingredients = []*Ingredient{
		ingredient0010,
	}

	return ingredient0010

}
