package mock

import (
	"github.com/ActiveState/cli/pkg/buildplan/raw"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/go-openapi/strfmt"
)

var BuildWithSourceFromStep = &raw.Build{
	Terminals: []*raw.NamedTarget{
		{
			Tag: "platform:00000000-0000-0000-0000-000000000001",
			// Step 1: Traversal starts here, this one points to an artifact
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
		},
	},
	Steps: []*raw.Step{
		{
			// Step 4: From here we can find which other nodes are linked to this one
			StepID:  "00000000-0000-0000-0000-000000000003",
			Outputs: []string{"00000000-0000-0000-0000-000000000002"},
			Inputs: []*raw.NamedTarget{
				// Step 5: Now we know which nodes are responsible for producing the output
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000004"}},
			},
		},
		{
			// Step 8: Same as step 4
			StepID:  "00000000-0000-0000-0000-000000000005",
			Outputs: []string{"00000000-0000-0000-0000-000000000004"},
			Inputs: []*raw.NamedTarget{
				// Step 9: Same as step 5
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000006"}},
			},
		},
	},
	Artifacts: []*raw.Artifact{
		{
			// Step 2: We got an artifact, but there may be more hiding behind this one
			NodeID:      "00000000-0000-0000-0000-000000000002",
			DisplayName: "installer",
			// Step 3: Now to traverse any other input nodes that generated this one, this goes to the step
			GeneratedBy: "00000000-0000-0000-0000-000000000003",
		},
		{
			// Step 6: We have another artifact, but since this is an x-artifact we also want meta info (ingredient name, version)
			NodeID:      "00000000-0000-0000-0000-000000000004",
			DisplayName: "pkgOne",
			// Step 7: Same as step 3
			GeneratedBy: "00000000-0000-0000-0000-000000000005",
		},
	},
	Sources: []*raw.Source{
		{
			// Step 10: We have our ingredient
			NodeID: "00000000-0000-0000-0000-000000000006",
		},
	},
}

var BuildWithSourceFromGeneratedBy = &raw.Build{
	Terminals: []*raw.NamedTarget{
		{
			Tag:     "platform:00000000-0000-0000-0000-000000000001",
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000004"},
		},
	},
	Artifacts: []*raw.Artifact{
		{
			NodeID:      "00000000-0000-0000-0000-000000000002",
			DisplayName: "installer",
			GeneratedBy: "00000000-0000-0000-0000-000000000004",
		},
		{
			NodeID:      "00000000-0000-0000-0000-000000000003",
			DisplayName: "installer",
			GeneratedBy: "00000000-0000-0000-0000-000000000004",
		},
	},
	Sources: []*raw.Source{
		{
			NodeID: "00000000-0000-0000-0000-000000000004",
		},
	},
}

var BuildWithBuildDeps = &raw.Build{
	Terminals: []*raw.NamedTarget{
		{
			Tag:     "platform:00000000-0000-0000-0000-000000000001",
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
		},
	},
	Steps: []*raw.Step{
		{
			StepID:  "00000000-0000-0000-0000-000000000003",
			Outputs: []string{"00000000-0000-0000-0000-000000000002"},
			Inputs: []*raw.NamedTarget{
				{Tag: "deps", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000004"}},
			},
		},
	},
	Artifacts: []*raw.Artifact{
		{
			NodeID:      "00000000-0000-0000-0000-000000000002",
			DisplayName: "installer",
			GeneratedBy: "00000000-0000-0000-0000-000000000003",
		},
		{
			NodeID:      "00000000-0000-0000-0000-000000000004",
			DisplayName: "pkgOne",
			GeneratedBy: "00000000-0000-0000-0000-000000000006",
		},
	},
	Sources: []*raw.Source{
		{
			NodeID: "00000000-0000-0000-0000-000000000006",
		},
	},
}

var BuildWithRuntimeDeps = &raw.Build{
	Terminals: []*raw.NamedTarget{
		{
			Tag:     "platform:00000000-0000-0000-0000-000000000001",
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
		},
	},
	Artifacts: []*raw.Artifact{
		{
			NodeID:      "00000000-0000-0000-0000-000000000002",
			DisplayName: "installer",
			MimeType:    types.XActiveStateArtifactMimeType,
			RuntimeDependencies: []strfmt.UUID{
				"00000000-0000-0000-0000-000000000007",
			},
			GeneratedBy: "00000000-0000-0000-0000-000000000003",
		},
		{
			NodeID:      "00000000-0000-0000-0000-000000000007",
			DisplayName: "pkgOne",
			MimeType:    types.XActiveStateArtifactMimeType,
			GeneratedBy: "00000000-0000-0000-0000-000000000008",
		},
	},
	Sources: []*raw.Source{
		{
			NodeID: "00000000-0000-0000-0000-000000000006",
		},
		{
			NodeID: "00000000-0000-0000-0000-000000000009",
		},
	},
}

// BuildWithInstallerDepsViaSrc is a build that includes an installer which has two artifacts as its dependencies.
var BuildWithInstallerDepsViaSrc = &raw.Build{
	Terminals: []*raw.NamedTarget{
		{
			Tag:     "platform:00000000-0000-0000-0000-000000000001",
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
		},
	},
	Steps: []*raw.Step{
		{
			StepID:  "00000000-0000-0000-0000-000000000003",
			Outputs: []string{"00000000-0000-0000-0000-000000000002"},
			Inputs: []*raw.NamedTarget{
				{
					Tag: "src", NodeIDs: []strfmt.UUID{
						"00000000-0000-0000-0000-000000000007",
						"00000000-0000-0000-0000-000000000010",
					},
				},
			},
		},
		{
			StepID:  "00000000-0000-0000-0000-000000000008",
			Outputs: []string{"00000000-0000-0000-0000-000000000007"},
			Inputs: []*raw.NamedTarget{
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000009"}},
			},
		},
		{
			StepID:  "00000000-0000-0000-0000-000000000011",
			Outputs: []string{"00000000-0000-0000-0000-000000000010"},
			Inputs: []*raw.NamedTarget{
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000012"}},
			},
		},
	},
	Artifacts: []*raw.Artifact{
		{
			NodeID:              "00000000-0000-0000-0000-000000000002",
			DisplayName:         "installer",
			MimeType:            "application/unrecognized",
			RuntimeDependencies: []strfmt.UUID{},
			GeneratedBy:         "00000000-0000-0000-0000-000000000003",
		},
		{
			NodeID:      "00000000-0000-0000-0000-000000000007",
			DisplayName: "pkgOne",
			MimeType:    types.XActiveStateArtifactMimeType,
			GeneratedBy: "00000000-0000-0000-0000-000000000008",
		},
		{
			NodeID:      "00000000-0000-0000-0000-000000000010",
			DisplayName: "pkgTwo",
			MimeType:    types.XActiveStateArtifactMimeType,
			GeneratedBy: "00000000-0000-0000-0000-000000000011",
		},
	},
	Sources: []*raw.Source{
		{
			"00000000-0000-0000-0000-000000000009",
			raw.IngredientSource{
				IngredientID: "00000000-0000-0000-0000-000000000009",
			},
		},
		{
			"00000000-0000-0000-0000-000000000012",
			raw.IngredientSource{
				IngredientID: "00000000-0000-0000-0000-000000000012",
			},
		},
	},
}

// BuildWithStateArtifactThroughPyWheel is a build with a state tool artifact that has a python wheel as its dependency
var BuildWithStateArtifactThroughPyWheel = &raw.Build{
	Terminals: []*raw.NamedTarget{
		{
			Tag:     "platform:00000000-0000-0000-0000-000000000001",
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
		},
	},
	Steps: []*raw.Step{
		{
			StepID:  "00000000-0000-0000-0000-000000000003",
			Outputs: []string{"00000000-0000-0000-0000-000000000002"},
			Inputs: []*raw.NamedTarget{
				{
					Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000007"},
				},
			},
		},
		{
			StepID:  "00000000-0000-0000-0000-000000000008",
			Outputs: []string{"00000000-0000-0000-0000-000000000007"},
			Inputs: []*raw.NamedTarget{
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000009"}},
			},
		},
	},
	Artifacts: []*raw.Artifact{
		{
			NodeID:      "00000000-0000-0000-0000-000000000002",
			DisplayName: "pkgStateArtifact",
			GeneratedBy: "00000000-0000-0000-0000-000000000003",
		},
		{
			NodeID:      "00000000-0000-0000-0000-000000000007",
			DisplayName: "pkgPyWheel",
			GeneratedBy: "00000000-0000-0000-0000-000000000008",
		},
	},
	Sources: []*raw.Source{
		{
			"00000000-0000-0000-0000-000000000009",
			raw.IngredientSource{
				IngredientID: "00000000-0000-0000-0000-000000000009",
			},
		},
	},
}

var BuildWithCommonRuntimeDepsViaSrc = &raw.Build{
	Terminals: []*raw.NamedTarget{
		{
			Tag:     "platform:00000000-0000-0000-0000-000000000001",
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
		},
	},
	Steps: []*raw.Step{
		{
			StepID:  "00000000-0000-0000-0000-000000000008",
			Outputs: []string{"00000000-0000-0000-0000-000000000007"},
			Inputs: []*raw.NamedTarget{
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000009"}},
			},
		},
		{
			StepID:  "00000000-0000-0000-0000-0000000000011",
			Outputs: []string{"00000000-0000-0000-0000-000000000010"},
			Inputs: []*raw.NamedTarget{
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000013"}},
			},
		},
		{
			StepID:  "00000000-0000-0000-0000-0000000000101",
			Outputs: []string{"00000000-0000-0000-0000-000000000100"},
			Inputs: []*raw.NamedTarget{
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000103"}},
			},
		},
	},
	Artifacts: []*raw.Artifact{
		{
			NodeID:      "00000000-0000-0000-0000-000000000007",
			DisplayName: "pkgOne",
			MimeType:    types.XActiveStateArtifactMimeType,
			GeneratedBy: "00000000-0000-0000-0000-000000000008",
			RuntimeDependencies: []strfmt.UUID{
				"00000000-0000-0000-0000-000000000100",
			},
		},
		{
			NodeID:      "00000000-0000-0000-0000-000000000010",
			DisplayName: "pkgTwo",
			MimeType:    types.XActiveStateArtifactMimeType,
			GeneratedBy: "00000000-0000-0000-0000-0000000000011",
			RuntimeDependencies: []strfmt.UUID{
				"00000000-0000-0000-0000-000000000100",
			},
		},
		{
			NodeID:      "00000000-0000-0000-0000-000000000100",
			DisplayName: "pkgThatsCommonDep",
			MimeType:    types.XActiveStateArtifactMimeType,
			GeneratedBy: "00000000-0000-0000-0000-0000000000101",
		},
	},
	Sources: []*raw.Source{
		{
			"00000000-0000-0000-0000-000000000009",
			raw.IngredientSource{
				IngredientID: "00000000-0000-0000-0000-000000000009",
			},
		},
		{
			"00000000-0000-0000-0000-000000000013",
			raw.IngredientSource{
				IngredientID: "00000000-0000-0000-0000-000000000013",
			},
		},
		{
			"00000000-0000-0000-0000-000000000103",
			raw.IngredientSource{
				IngredientID: "00000000-0000-0000-0000-000000000103",
			},
		},
	},
}

// BuildWithRuntimeDepsViaSrcCycle is a build with a cycle in the runtime dependencies.
// The cycle is as follows:
// 00000000-0000-0000-0000-000000000002 (Terminal Artifact)
//
//	-> 00000000-0000-0000-0000-000000000003 (Generated by Step)
//	  -> 00000000-0000-0000-0000-000000000007 (Step Input Artifact)
//	    -> 00000000-0000-0000-0000-000000000008 (Generated by Step)
//	      -> 00000000-0000-0000-0000-000000000010 (Step Input Artifact)
//	        -> 00000000-0000-0000-0000-000000000011 (Generated by Step)
//	          -> 00000000-0000-0000-0000-000000000013 (Step Input Artifact)
//	            -> 00000000-0000-0000-0000-000000000002 (Runtime dependency Artifact - Generates Cycle)
var BuildWithRuntimeDepsViaSrcCycle = &raw.Build{
	Terminals: []*raw.NamedTarget{
		{
			Tag:     "platform:00000000-0000-0000-0000-000000000001",
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
		},
	},
	Steps: []*raw.Step{
		{
			StepID:  "00000000-0000-0000-0000-000000000003",
			Outputs: []string{"00000000-0000-0000-0000-000000000002"},
			Inputs: []*raw.NamedTarget{
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000007"}},
			},
		},
		{
			StepID:  "00000000-0000-0000-0000-000000000008",
			Outputs: []string{"00000000-0000-0000-0000-000000000007"},
			Inputs: []*raw.NamedTarget{
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000010"}},
			},
		},
		{
			StepID: "00000000-0000-0000-0000-000000000011",
			Outputs: []string{
				"00000000-0000-0000-0000-000000000010",
			},
			Inputs: []*raw.NamedTarget{
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000013"}},
			},
		},
	},
	Artifacts: []*raw.Artifact{
		{
			NodeID:      "00000000-0000-0000-0000-000000000002",
			DisplayName: "installer",
			MimeType:    "application/unrecognized",
			GeneratedBy: "00000000-0000-0000-0000-000000000003",
		},
		{
			NodeID:      "00000000-0000-0000-0000-000000000007",
			DisplayName: "pkgOne",
			MimeType:    "application/unrecognized",
			GeneratedBy: "00000000-0000-0000-0000-000000000008",
		},
		{
			NodeID:      "00000000-0000-0000-0000-000000000010",
			DisplayName: "pkgTwo",
			MimeType:    "application/unrecognized",
			GeneratedBy: "00000000-0000-0000-0000-000000000011",
		},
		{
			NodeID:              "00000000-0000-0000-0000-000000000013",
			DisplayName:         "pkgThree",
			MimeType:            types.XActiveStateArtifactMimeType,
			RuntimeDependencies: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"}, // Generates a cycle back to the first artifact
			GeneratedBy:         "00000000-0000-0000-0000-000000000011",
		},
	},
	Sources: []*raw.Source{
		{
			NodeID: "00000000-0000-0000-0000-000000000006",
		},
		{
			NodeID: "00000000-0000-0000-0000-000000000009",
		},
		{
			NodeID: "00000000-0000-0000-0000-000000000012",
		},
	},
}
