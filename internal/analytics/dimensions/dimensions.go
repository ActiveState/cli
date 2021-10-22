package dimensions

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/internal/singleton/uniqid"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/sysinfo"
	"github.com/imdario/mergo"
)

type Map struct {
	Version          *string
	BranchName       *string
	UserID           *string
	OSName           *string
	OSVersion        *string
	InstallSource    *string
	MachineID        *string
	UniqID           *string
	SessionToken     *string
	UpdateTag        *string
	ProjectNameSpace *string
	OutputType       *string
	ProjectID        *string
}

func NewDefaultDimensions(pjNamespace, sessionToken, updateTag string) *Map {
	installSource, err := storage.InstallSource()
	if err != nil {
		logging.Error("Could not detect installSource: %s", errs.Join(err, " :: ").Error())
	}

	machineID := machineid.UniqID()
	if machineID == machineid.UnknownID || machineID == machineid.FallbackID {
		logging.Error("unknown machine id: %s", machineID)
	}
	deviceID := uniqid.Text()

	var userIDString string
	userID := authentication.LegacyGet().UserID()
	if userID != nil {
		userIDString = userID.String()
	}

	osName := sysinfo.OS().String()
	osVersion := "unknown"
	osvInfo, err := sysinfo.OSVersion()
	if err != nil {
		logging.Errorf("Could not detect osVersion: %v", err)
	}
	if osvInfo != nil {
		osVersion = osvInfo.Version
	}

	return &Map{
		p.StrP(constants.Version),
		p.StrP(constants.BranchName),
		p.StrP(userIDString),
		p.StrP(osName),
		p.StrP(osVersion),
		p.StrP(installSource),
		p.StrP(machineID),
		p.StrP(deviceID),
		p.StrP(sessionToken),
		p.StrP(updateTag),
		p.StrP(pjNamespace),
		p.StrP(string(output.PlainFormatName)),
		p.StrP(""),
	}
}

// WithClientData returns a copy of the custom dimensions struct with client-specific fields overwritten
func (d *Map) WithClientData(projectNameSpace, output, userID string) *Map {
	res := *d
	res.ProjectNameSpace = p.StrP(projectNameSpace)
	res.OutputType = p.StrP(output)
	res.UserID = p.StrP(userID)
	return &res
}

func (m *Map) Merge(mergeWith ...*Map) {
	for _, dim := range mergeWith {
		if err := mergo.Merge(m, dim); err != nil {
			logging.Critical("Could not merge dimension maps: %s", errs.JoinMessage(err))
		}
	}
}

func (d *Map) ToMap() map[string]string {
	return map[string]string{
		// Commented out idx 1 so it's clear why we start with 2. We used to log the hostname while dogfooding internally.
		// "1": "hostname (deprected)"
		"2":  p.PStr(d.Version),
		"3":  p.PStr(d.BranchName),
		"4":  p.PStr(d.UserID),
		"5":  p.PStr(d.OutputType),
		"6":  p.PStr(d.OSName),
		"7":  p.PStr(d.OSVersion),
		"8":  p.PStr(d.InstallSource),
		"9":  p.PStr(d.MachineID),
		"10": p.PStr(d.ProjectNameSpace),
		"11": p.PStr(d.SessionToken),
		"12": p.PStr(d.UniqID),
		"13": p.PStr(d.UpdateTag),
		"14": p.PStr(d.ProjectID),
	}
}
