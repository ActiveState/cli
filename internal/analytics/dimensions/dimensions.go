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
	"github.com/ActiveState/cli/internal/ptr"
	"github.com/ActiveState/cli/internal/rollbar"
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
		ptr.To(constants.Version),
		ptr.To(constants.BranchName),
		ptr.To(userIDString),
		ptr.To(osName),
		ptr.To(osVersion),
		ptr.To(installSource),
		ptr.To(deviceID),
		ptr.To(sessionToken),
		ptr.To(updateTag),
		ptr.To(pjNamespace),
		ptr.To(string(output.PlainFormatName)),
		ptr.To(""),
		ptr.To(CalculateFlags()),
		ptr.To(""),
		ptr.To(""),
		ptr.To(instanceid.ID()),
		ptr.To(""),
		ptr.To(osutils.ExecutableName()),
		ptr.To(0),
		nil,
	}
}

func (v *Values) Clone() *Values {
	return &Values{
		Version:          ptr.Renew(v.Version),
		BranchName:       ptr.Renew(v.BranchName),
		UserID:           ptr.Renew(v.UserID),
		OSName:           ptr.Renew(v.OSName),
		OSVersion:        ptr.Renew(v.OSVersion),
		InstallSource:    ptr.Renew(v.InstallSource),
		UniqID:           ptr.Renew(v.UniqID),
		SessionToken:     ptr.Renew(v.SessionToken),
		UpdateTag:        ptr.Renew(v.UpdateTag),
		ProjectNameSpace: ptr.Renew(v.ProjectNameSpace),
		OutputType:       ptr.Renew(v.OutputType),
		ProjectID:        ptr.Renew(v.ProjectID),
		Flags:            ptr.Renew(v.Flags),
		Trigger:          ptr.Renew(v.Trigger),
		Headless:         ptr.Renew(v.Headless),
		InstanceID:       ptr.Renew(v.InstanceID),
		CommitID:         ptr.Renew(v.CommitID),
		Command:          ptr.Renew(v.Command),
		Sequence:         ptr.Renew(v.Sequence),
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

	if ptr.Deref(v.UniqID) == "" {
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
