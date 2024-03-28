package model

import (
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

func (bs Branches) ToMonoBranches() mono_models.Branches {
	if bs == nil {
		return nil
	}

	var mbs mono_models.Branches
	for _, b := range bs {
		mbs = append(mbs, b.ToMonoBranch())
	}

	return mbs
}

func (b *Branch) ToMonoBranch() *mono_models.Branch {
	return &mono_models.Branch{
		BranchID:     b.BranchID,
		CommitID:     b.CommitID,
		Default:      ptrBoolToBool(b.Main),
		Label:        b.Label,
		ProjectID:    *b.ProjectID, // potential to orphan?
		TrackingType: b.TrackingType,
		Tracks:       b.Tracks,
	}
}

func (p *Project) ToMonoProject() (*mono_models.Project, error) {
	for _, b := range p.Branches {
		if b.ProjectID == nil {
			return nil, errs.New("branch does not have project ID")
		}
	}

	mp := mono_models.Project{
		Added:          makeStrfmtDateTime(p.Added),
		Branches:       p.Branches.ToMonoBranches(),
		CreatedBy:      ptrStrfmtUUIDToPtrString(p.CreatedBy),
		Description:    p.Description,
		ForkedFrom:     forkedProjectToMonoForkedFrom(p.ForkedProject),
		LastEdited:     makeStrfmtDateTime(p.Changed),
		Managed:        p.Managed,
		Name:           p.Name,
		OrganizationID: p.OrganizationID,
		Private:        p.Private,
		ProjectID:      p.ProjectID,
		RepoURL:        p.RepoURL,
	}

	return &mp, nil
}

func makeStrfmtDateTime(t Time) strfmt.DateTime {
	dt, err := strfmt.ParseDateTime(t.Time.Format(time.RFC3339))
	if err != nil {
		panic(err) // this should never happen
	}
	return dt
}

func forkedProjectToMonoForkedFrom(fp *ForkedProject) *mono_models.ProjectForkedFrom {
	if fp == nil {
		return nil
	}
	return &mono_models.ProjectForkedFrom{
		Project:      fp.Name,
		Organization: fp.Organization.URLName,
	}
}

func ptrStrfmtUUIDToPtrString(id *strfmt.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

func ptrBoolToBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
