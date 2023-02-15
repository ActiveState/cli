package dimensions

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/internal/singleton/uniqid"
	analytics2 "github.com/ActiveState/cli/pkg/platform/analytics"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/sysinfo"
)

func NewDefaultDimensions(pjNamespace, sessionToken, updateTag string) *analytics2.Dimensions {
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

	return &analytics2.Dimensions{
		p.StrP(constants.Version),
		p.StrP(constants.BranchName),
		p.StrP(userIDString),
		p.StrP(osName),
		p.StrP(osVersion),
		p.StrP(installSource),
		p.StrP(deviceID),
		p.StrP(sessionToken),
		p.StrP(updateTag),
		p.StrP(pjNamespace),
		p.StrP(string(output.PlainFormatName)),
		p.StrP(""),
		p.StrP(analytics.CalculateFlags()),
		p.StrP(""),
		p.StrP(""),
		p.StrP(instanceid.ID()),
		p.StrP(""),
		p.StrP(osutils.ExecutableName()),
		p.IntP(0),
		nil,
	}
}
