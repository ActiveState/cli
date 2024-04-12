package buildplan

import (
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
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
	Type                 string                 `json:"__typename"`
	BuildPlanID          strfmt.UUID            `json:"buildPlanID"`
	Status               string                 `json:"status"`
	Terminals            []*NamedTarget         `json:"terminals"`
	Artifacts            []*Artifact            `json:"artifacts"`
	Steps                []*Step                `json:"steps"`
	Sources              []*Source              `json:"sources"`
	BuildLogIDs          []*BuildLogID          `json:"buildLogIds"`
	ResolvedRequirements []*ResolvedRequirement `json:"resolvedRequirements"`
	*response.Error
	*response.PlanningError
}

// BuildLogID is the ID used to initiate a connection with the BuildLogStreamer.
type BuildLogID struct {
	ID         string      `json:"id"`
	PlatformID strfmt.UUID `json:"platformID"`
}

// NamedTarget is a special target used for terminals.
type NamedTarget struct {
	Tag     string        `json:"tag"`
	NodeIDs []strfmt.UUID `json:"nodeIds"`
}

// Artifact represents a downloadable artifact.
// This artifact may or may not be installable by the State Tool.
type Artifact struct {
	Type                string        `json:"__typename"`
	NodeID              strfmt.UUID   `json:"nodeId"`
	DisplayName         string        `json:"displayName"`
	MimeType            string        `json:"mimeType"`
	GeneratedBy         strfmt.UUID   `json:"generatedBy"`
	RuntimeDependencies []strfmt.UUID `json:"runtimeDependencies"`
	Status              string        `json:"status"`
	URL                 string        `json:"url"`
	LogURL              string        `json:"logURL"`
	Checksum            string        `json:"checksum"`

	// Error fields
	Errors      []string `json:"errors"`
	Attempts    float64  `json:"attempts"`
	NextAttempt string   `json:"nextAttempt"`
}

// Step represents a single step in the build plan.
// A step takes some input, processes it, and produces some output.
// This is usually a build step. The input represents a set of target
// IDs and the output are a set of artifact IDs.
type Step struct {
	StepID  strfmt.UUID    `json:"stepId"`
	Inputs  []*NamedTarget `json:"inputs"`
	Outputs []string       `json:"outputs"`
}

// Source represents the source of an artifact.
type Source struct {
	NodeID    strfmt.UUID `json:"nodeId"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace"`
	Version   string      `json:"version"`
}

type ResolvedRequirement struct {
	Requirement *response.Requirement `json:"requirement"`
	Source      strfmt.UUID           `json:"resolvedSource"`
}
