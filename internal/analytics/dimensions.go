package analytics

import (
	"encoding/json"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils/p"
)

type Dimensions struct {
	Version          *string
	BranchName       *string
	UserID           *string
	OSName           *string
	OSVersion        *string
	InstallSource    *string
	UniqID           *string
	SessionToken     *string
	UpdateTag        *string
	ProjectNameSpace *string
	OutputType       *string
	ProjectID        *string
	Flags            *string
	Trigger          *string
	Headless         *string
	InstanceID       *string
	CommitID         *string
	Command          *string
	Sequence         *int

	PreProcessor func(*Dimensions) error
}

func (v *Dimensions) Clone() *Dimensions {
	return &Dimensions{
		Version:          p.PstrP(v.Version),
		BranchName:       p.PstrP(v.BranchName),
		UserID:           p.PstrP(v.UserID),
		OSName:           p.PstrP(v.OSName),
		OSVersion:        p.PstrP(v.OSVersion),
		InstallSource:    p.PstrP(v.InstallSource),
		UniqID:           p.PstrP(v.UniqID),
		SessionToken:     p.PstrP(v.SessionToken),
		UpdateTag:        p.PstrP(v.UpdateTag),
		ProjectNameSpace: p.PstrP(v.ProjectNameSpace),
		OutputType:       p.PstrP(v.OutputType),
		ProjectID:        p.PstrP(v.ProjectID),
		Flags:            p.PstrP(v.Flags),
		Trigger:          p.PstrP(v.Trigger),
		Headless:         p.PstrP(v.Headless),
		InstanceID:       p.PstrP(v.InstanceID),
		CommitID:         p.PstrP(v.CommitID),
		Command:          p.PstrP(v.Command),
		Sequence:         p.PintP(v.Sequence),
		PreProcessor:     v.PreProcessor,
	}
}

func (m *Dimensions) Merge(mergeWith ...*Dimensions) {
	// This is awkward and long, but using mergo was not an option here because it cannot differentiate between
	// falsy values and nil pointers
	for _, dim := range mergeWith {
		if dim.Version != nil {
			m.Version = dim.Version
		}
		if dim.BranchName != nil {
			m.BranchName = dim.BranchName
		}
		if dim.UserID != nil {
			m.UserID = dim.UserID
		}
		if dim.OSName != nil {
			m.OSName = dim.OSName
		}
		if dim.OSVersion != nil {
			m.OSVersion = dim.OSVersion
		}
		if dim.InstallSource != nil {
			m.InstallSource = dim.InstallSource
		}
		if dim.UniqID != nil {
			m.UniqID = dim.UniqID
		}
		if dim.SessionToken != nil {
			m.SessionToken = dim.SessionToken
		}
		if dim.UpdateTag != nil {
			m.UpdateTag = dim.UpdateTag
		}
		if dim.ProjectNameSpace != nil {
			m.ProjectNameSpace = dim.ProjectNameSpace
		}
		if dim.OutputType != nil {
			m.OutputType = dim.OutputType
		}
		if dim.ProjectID != nil {
			m.ProjectID = dim.ProjectID
		}
		if dim.Flags != nil {
			m.Flags = dim.Flags
		}
		if dim.Trigger != nil {
			m.Trigger = dim.Trigger
		}
		if dim.Headless != nil {
			m.Headless = dim.Headless
		}
		if dim.InstanceID != nil {
			m.InstanceID = dim.InstanceID
		}
		if dim.CommitID != nil {
			m.CommitID = dim.CommitID
		}
		if dim.Command != nil {
			m.Command = dim.Command
		}
		if dim.Sequence != nil {
			m.Sequence = dim.Sequence
		}
		if dim.PreProcessor != nil {
			m.PreProcessor = dim.PreProcessor
		}
	}
}

func (v *Dimensions) RegisterPreProcessor(f func(*Dimensions) error) {
	v.PreProcessor = f
}

func (v *Dimensions) PreProcess() error {
	if v.PreProcessor != nil {
		if err := v.PreProcessor(v); err != nil {
			return errs.Wrap(err, "PreProcessor failed: %s", errs.JoinMessage(err))
		}
	}

	if p.PStr(v.UniqID) == "" {
		return errs.New("device id is unset when creating analytics event")
	}

	return nil
}

func (v *Dimensions) Marshal() (string, error) {
	dimMarshalled, err := json.Marshal(v)
	if err != nil {
		return "", errs.Wrap(err, "Could not marshal dimensions")
	}
	return string(dimMarshalled), nil
}
