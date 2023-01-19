package model

import (
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/go-openapi/strfmt"
)

type Requirement struct {
	mono_models.Checkpoint
	VersionConstraints mono_models.Constraints `json:"constraint_json,omitempty"`
	CommitID           strfmt.UUID             `json:"commit_id"`
}

type Commit struct {
	AtTime strfmt.DateTime `json:"at_time"`
}

type Checkpoint struct {
	Requirements []*Requirement `json:"vcs_checkpoints"`
	Commit       *Commit        `json:"vcs_commits_by_pk"`
}

