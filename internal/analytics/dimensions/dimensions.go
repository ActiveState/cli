package dimensions

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/singleton/uniqid"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/sysinfo"
)

type Values struct {
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

	preProcessor func(*Values) error
}

func NewDefaultDimensions(pjNamespace, sessionToken, updateTag string) *Values {
	installSource, err := storage.InstallSource()
	if err != nil {
		multilog.Error("Could not detect installSource: %s", errs.Join(err, " :: ").Error())
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
		multilog.Log(logging.ErrorNoStacktrace, rollbar.Error)("Could not detect osVersion: %v", err)
	}
	if osvInfo != nil {
		osVersion = osvInfo.Version
	}

	return &Values{
		ptr.StrP(constants.Version),
		ptr.StrP(constants.BranchName),
		ptr.StrP(userIDString),
		ptr.StrP(osName),
		ptr.StrP(osVersion),
		ptr.StrP(installSource),
		ptr.StrP(deviceID),
		ptr.StrP(sessionToken),
		ptr.StrP(updateTag),
		ptr.StrP(pjNamespace),
		ptr.StrP(string(output.PlainFormatName)),
		ptr.StrP(""),
		ptr.StrP(CalculateFlags()),
		ptr.StrP(""),
		ptr.StrP(""),
		ptr.StrP(instanceid.ID()),
		ptr.StrP(""),
		ptr.StrP(osutils.ExecutableName()),
		ptr.IntP(0),
		nil,
	}
}

func (v *Values) Clone() *Values {
	return &Values{
		Version:          ptr.PstrP(v.Version),
		BranchName:       ptr.PstrP(v.BranchName),
		UserID:           ptr.PstrP(v.UserID),
		OSName:           ptr.PstrP(v.OSName),
		OSVersion:        ptr.PstrP(v.OSVersion),
		InstallSource:    ptr.PstrP(v.InstallSource),
		UniqID:           ptr.PstrP(v.UniqID),
		SessionToken:     ptr.PstrP(v.SessionToken),
		UpdateTag:        ptr.PstrP(v.UpdateTag),
		ProjectNameSpace: ptr.PstrP(v.ProjectNameSpace),
		OutputType:       ptr.PstrP(v.OutputType),
		ProjectID:        ptr.PstrP(v.ProjectID),
		Flags:            ptr.PstrP(v.Flags),
		Trigger:          ptr.PstrP(v.Trigger),
		Headless:         ptr.PstrP(v.Headless),
		InstanceID:       ptr.PstrP(v.InstanceID),
		CommitID:         ptr.PstrP(v.CommitID),
		Command:          ptr.PstrP(v.Command),
		Sequence:         ptr.PintP(v.Sequence),
		preProcessor:     v.preProcessor,
	}
}

func (m *Values) Merge(mergeWith ...*Values) {
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
		if dim.preProcessor != nil {
			m.preProcessor = dim.preProcessor
		}
	}
}

func (v *Values) RegisterPreProcessor(f func(*Values) error) {
	v.preProcessor = f
}

func (v *Values) PreProcess() error {
	if v.preProcessor != nil {
		if err := v.preProcessor(v); err != nil {
			return errs.Wrap(err, "PreProcessor failed: %s", errs.JoinMessage(err))
		}
	}

	if ptr.PStr(v.UniqID) == "" {
		return errs.New("device id is unset when creating analytics event")
	}

	return nil
}

func (v *Values) Marshal() (string, error) {
	dimMarshalled, err := json.Marshal(v)
	if err != nil {
		return "", errs.Wrap(err, "Could not marshal dimensions")
	}
	return string(dimMarshalled), nil
}

func CalculateFlags() string {
	flags := []string{}
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
		}
	}
	return strings.Join(flags, " ")
}
