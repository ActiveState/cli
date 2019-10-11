package client

import "github.com/ActiveState/cli/internal/platform/api/model"

type ProjectProvider interface {
	ProjectByOrgAndName(org, name string) (*ProjectResp, error)
}

type ProjectsResp struct {
	Projects []*model.Project `json:"projects"`
}

type ProjectResp struct {
	Project *model.Project `json:"project"`
}

func (psr *ProjectsResp) ProjectToProjectResp(index int) (*ProjectResp, error) {
	if psr.Projects == nil || index < 0 || len(psr.Projects) < index+1 {
		return nil, ErrNoValueAvailable
	}

	return &ProjectResp{Project: psr.Projects[index]}, nil
}
