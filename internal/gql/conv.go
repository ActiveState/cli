package gql

import (
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

func (psr *ProjectsResp) FirstToProjectResp() (*ProjectResp, error) {
	if psr.Projects == nil || len(psr.Projects) == 0 {
		return nil, ErrNoValueAvailable
	}

	return &ProjectResp{Project: psr.Projects[0]}, nil
}

func (pr *ProjectResp) ToMonoProject() (*mono_models.Project, error) {
	if pr == nil {
		return nil, ErrNoValueAvailable
	}

	p := pr.Project

	for _, b := range p.Branches {
		if b.ProjectID == nil {
			return nil, ErrMissingBranchProjectID
		}
	}

	mp := mono_models.Project{
		Added:          makeStrfmtDateTime(p.Added),
		Branches:       p.Branches.ToMonoBranches(),
		CreatedBy:      ptrStrfmtUUIDToPtrString(p.CreatedBy),
		Description:    p.Description,
		ForkedFrom:     forkedProjectToMonoForkedFrom(p.ForkedProject),
		Languages:      nil,
		LastEdited:     makeStrfmtDateTime(p.Changed),
		Managed:        p.Managed,
		Name:           p.Name,
		OrganizationID: p.OrganizationID,
		Platforms:      nil,
		Private:        p.Private,
		ProjectID:      p.ProjectID,
		RepoURL:        newStrfmtURI(p.RepoURL),
	}

	return &mp, nil
}
