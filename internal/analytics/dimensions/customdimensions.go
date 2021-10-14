package dimensions

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/singleton/uniqid"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/sysinfo"
)

type Map struct {
	Version          string
	BranchName       string
	UserID           string
	OSName           string
	OSVersion        string
	InstallSource    string
	MachineID        string
	UniqID           string
	SessionToken     string
	UpdateTag        string
	ProjectNameSpace string
	OutputType       string
	ProjectID        string
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
		constants.Version,
		constants.BranchName,
		userIDString,
		osName,
		osVersion,
		installSource,
		machineID,
		deviceID,
		sessionToken,
		updateTag,
		pjNamespace,
		string(output.PlainFormatName),
		"",
	}
}

// WithClientData returns a copy of the custom dimensions struct with client-specific fields overwritten
func (d *Map) WithClientData(projectNameSpace, output, userID string) *Map {
	res := *d
	res.ProjectNameSpace = projectNameSpace
	res.OutputType = output
	res.UserID = userID
	return &res
}

func (d *Map) ToMap() map[string]string {
	return map[string]string{
		// Commented out idx 1 so it's clear why we start with 2. We used to log the hostname while dogfooding internally.
		// "1": "hostname (deprected)"
		"2":  d.Version,
		"3":  d.BranchName,
		"4":  d.UserID,
		"5":  d.OutputType,
		"6":  d.OSName,
		"7":  d.OSVersion,
		"8":  d.InstallSource,
		"9":  d.MachineID,
		"10": d.ProjectNameSpace,
		"11": d.SessionToken,
		"12": d.UniqID,
		"13": d.UpdateTag,
		"14": d.ProjectID,
	}
}
