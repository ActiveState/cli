package raw

import (
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/go-openapi/strfmt"
)

var buildWithSourceFromStep = &Build{
	Terminals: []*NamedTarget{
		{
			Tag: "platform:00000000-0000-0000-0000-000000000001",
			// Step 1: Traversal starts here, this one points to an artifact
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
		},
	},
	Steps: []*Step{
		{
			// Step 4: From here we can find which other nodes are linked to this one
			StepID:  "00000000-0000-0000-0000-000000000003",
			Outputs: []string{"00000000-0000-0000-0000-000000000002"},
			Inputs: []*NamedTarget{
				// Step 5: Now we know which nodes are responsible for producing the output
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000004"}},
			},
		},
		{
			// Step 8: Same as step 4
			StepID:  "00000000-0000-0000-0000-000000000005",
			Outputs: []string{"00000000-0000-0000-0000-000000000004"},
			Inputs: []*NamedTarget{
				// Step 9: Same as step 5
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000006"}},
			},
		},
	},
	Artifacts: []*Artifact{
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
	Sources: []*Source{
		{
			// Step 10: We have our ingredient
			NodeID: "00000000-0000-0000-0000-000000000006",
		},
	},
}

var buildWithSourceFromGeneratedBy = &Build{
	Terminals: []*NamedTarget{
		{
			Tag:     "platform:00000000-0000-0000-0000-000000000001",
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000004"},
		},
	},
	Artifacts: []*Artifact{
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
	Sources: []*Source{
		{
			NodeID: "00000000-0000-0000-0000-000000000004",
		},
	},
}

var buildWithBuildDeps = &Build{
	Terminals: []*NamedTarget{
		{
			Tag:     "platform:00000000-0000-0000-0000-000000000001",
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
		},
	},
	Steps: []*Step{
		{
			StepID:  "00000000-0000-0000-0000-000000000003",
			Outputs: []string{"00000000-0000-0000-0000-000000000002"},
			Inputs: []*NamedTarget{
				{Tag: "deps", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000004"}},
			},
		},
	},
	Artifacts: []*Artifact{
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
	Sources: []*Source{
		{
			NodeID: "00000000-0000-0000-0000-000000000006",
		},
	},
}

var buildWithRuntimeDeps = &Build{
	Terminals: []*NamedTarget{
		{
			Tag:     "platform:00000000-0000-0000-0000-000000000001",
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
		},
	},
	Artifacts: []*Artifact{
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
	Sources: []*Source{
		{
			NodeID: "00000000-0000-0000-0000-000000000006",
		},
		{
			NodeID: "00000000-0000-0000-0000-000000000009",
		},
	},
}

var buildWithRuntimeDepsViaSrc = &Build{
	Terminals: []*NamedTarget{
		{
			Tag:     "platform:00000000-0000-0000-0000-000000000001",
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
		},
	},
	Steps: []*Step{
		{
			StepID:  "00000000-0000-0000-0000-000000000003",
			Outputs: []string{"00000000-0000-0000-0000-000000000002"},
			Inputs: []*NamedTarget{
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000007"}},
			},
		},
	},
	Artifacts: []*Artifact{
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
	},
	Sources: []*Source{
		{
			NodeID: "00000000-0000-0000-0000-000000000006",
		},
		{
			NodeID: "00000000-0000-0000-0000-000000000009",
		},
	},
}

var buildWithRuntimeDepsViaSrcCycle = &Build{
	Terminals: []*NamedTarget{
		{
			Tag:     "platform:00000000-0000-0000-0000-000000000001",
			NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000002"},
		},
	},
	Steps: []*Step{
		{
			StepID:  "00000000-0000-0000-0000-000000000003",
			Outputs: []string{"00000000-0000-0000-0000-000000000002"},
			Inputs: []*NamedTarget{
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000007"}},
			},
		},
		{
			StepID:  "00000000-0000-0000-0000-000000000008",
			Outputs: []string{"00000000-0000-0000-0000-000000000007"},
			Inputs: []*NamedTarget{
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000010"}},
			},
		},
		{
			StepID: "00000000-0000-0000-0000-000000000011",
			Outputs: []string{
				"00000000-0000-0000-0000-000000000010",
			},
			Inputs: []*NamedTarget{
				{Tag: "src", NodeIDs: []strfmt.UUID{"00000000-0000-0000-0000-000000000013"}},
			},
		},
	},
	Artifacts: []*Artifact{
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
			RuntimeDependencies: []strfmt.UUID{"00000000-0000-0000-0000-000000000010"},
			GeneratedBy:         "00000000-0000-0000-0000-000000000011",
		},
	},
	Sources: []*Source{
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
