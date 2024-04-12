package response

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/go-openapi/strfmt"
)

// Build is a directed acyclic graph. It begins with a set of terminal nodes
// that resolve to artifacts via a set of steps.
// The expected format of a build plan is:
//
//	{
//	    "build": {
//	        "__typename": "BuildReady",
//	        "buildLogIds": [
//	            {
//	                "id": "1f717bf7-3573-5144-834b-75917dd8f60c",
//	                "type": "RECIPE_ID",
//	                "platformId": ""
//	            }
//	        ],
//	        "status": "READY",
//	        "terminals": [
//	            {
//	                "tag": "platform:78977bc8-0f32-519d-80f3-9043f059398c",
//	                "targetIDs": [
//	                    "311aacc7-a596-59c3-bbc9-cf2340721136",
//	                    "e02c6998-5357-5bc5-a785-6bd890a4af46"
//	                ]
//	            }
//	        ],
//	        "sources": [
//	            {
//	                "targetID": "6c91bc10-e8e2-50a6-8cca-ebd3f1e3f549",
//	                "name": "zlib",
//	                "namespace": "shared",
//	                "version": "1.2.13"
//	            },
//	            ...
//	        ],
//	        "steps": [
//	            {
//	                "targetID": "ab276a34-0b24-51b5-aacc-7323442f59ad",
//	                "inputs": [
//	                    {
//	                        "tag": "builder",
//	                        "targetIDs": [
//	                            "357d394b-6ce6-5385-be81-1754348fe5dd"
//	                        ]
//	                    },
//	                    {
//	                        "tag": "src",
//	                        "targetIDs": [
//	                            "bd5232a0-55de-52bd-ba29-1c58b9072232"
//	                        ]
//	                    },
//	                    {
//	                        "tag": "deps",
//	                        "targetIDs": []
//	                    }
//	                ],
//	                "outputs": [
//	                    "3ca4edd7-7746-55a1-a3cb-15b41b83ae52"
//	                ]
//	            },
//	            ...
//	        ],
//	        "artifacts": [
//	            {
//	                "__typename": "ArtifactSucceeded",
//	                "targetID": "7322308b-9789-50eb-b843-446cca78d855",
//	                "mimeType": "application/x-activestate-builder",
//	                "generatedBy": "8e5a488c-25b4-54b6-adfb-9d66d60f449f",
//	                "runtimeDependencies": [
//	                    "9a02d063-e3b6-5230-8cbe-f8769ced5a06",
//	                    "f9c838fc-e477-5f39-9cfc-3ffa804b4d53",
//	                    "b04ea3ed-9632-5e59-a571-201cfc225d36",
//	                    "2c64301a-9789-5cc3-b9b6-011bc7554268"
//	                ],
//	                "status": "SUCCEEDED",
//	                "logURL": "",
//	                "url": "s3://platform-sources/builder/0705c78c125b8b0f30e7fa6aeb30ac5f71c99511df40a6b62223be528f89385d/wheel-builder-lib.tar.gz",
//	                "checksum": "0705c78c125b8b0f30e7fa6aeb30ac5f71c99511df40a6b62223be528f89385d"
//	            },
//	            ...
//	        ]
//	    }
//	}
type Build struct {
	Type                 string                       `json:"__typename"`
	BuildPlanID          strfmt.UUID                  `json:"buildPlanID"`
	Status               string                       `json:"status"`
	Terminals            []*types.NamedTarget         `json:"terminals"`
	Artifacts            []*types.Artifact            `json:"artifacts"`
	Steps                []*types.Step                `json:"steps"`
	Sources              []*types.Source              `json:"sources"`
	BuildLogIDs          []*types.BuildLogID          `json:"buildLogIds"`
	ResolvedRequirements []*types.ResolvedRequirement `json:"resolvedRequirements"`
	*Error
	*PlanningError
}

func (b *Build) Engine() types.BuildEngine {
	buildEngine := types.Alternative
	for _, s := range b.Sources {
		if s.Namespace == "builder" && s.Name == "camel" {
			buildEngine = types.Camel
			break
		}
	}
	return buildEngine
}

// RecipeID extracts the recipe ID from the BuildLogIDs.
// We do this because if the build is in progress we will need to reciepe ID to
// initialize the build log streamer.
// This information will only be populated if the build is an alternate build.
// This is specified in the build planner queries.
func (b *Build) RecipeID() (strfmt.UUID, error) {
	var result strfmt.UUID
	for _, id := range b.BuildLogIDs {
		if result != "" && result.String() != id.ID {
			return result, errs.New("Build plan contains multiple recipe IDs")
		}
		result = strfmt.UUID(id.ID)
	}
	return result, nil
}

func (b *Build) OrderedArtifacts() []strfmt.UUID {
	res := make([]strfmt.UUID, 0, len(b.Artifacts))
	for _, a := range b.Artifacts {
		res = append(res, a.NodeID)
	}
	return res
}

func (b *Build) Ready() bool {
	return b.Status == types.Completed
}
