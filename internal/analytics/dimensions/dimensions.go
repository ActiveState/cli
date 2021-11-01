package dimensions

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/internal/singleton/uniqid"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/sysinfo"
	"github.com/imdario/mergo"
)

type Values struct {
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
	Flags            *string
	Trigger          *string
	Headless         *string
	InstanceID       *string

	preProcessor func(*Values) error
}

func NewDefaultDimensions(pjNamespace, sessionToken, updateTag string) *Values {
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

	return &Values{
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
		p.StrP(CalculateFlags()),
		p.StrP(""),
		p.StrP(""),
		p.StrP(instanceid.ID()),
		nil,
	}
}

func (m *Values) Merge(mergeWith ...*Values) {
	for _, dim := range mergeWith {
		if err := mergo.Merge(m, dim, mergo.WithOverride); err != nil {
			logging.Critical("Could not merge dimension maps: %s", errs.JoinMessage(err))
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

	if p.PStr(v.UniqID) == machineid.FallbackID {
		return errs.New("machine id was set to fallback id when creating analytics event")
	}

	return nil
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
