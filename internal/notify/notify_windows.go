package notify

import (
	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"gopkg.in/toast.v1"
)

func Send(title, message, actionName, actionLink string) error {
	iconBytes, err := assets.ReadFileBytes("icon.png")
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	iconFile, err := fileutils.WriteTempFile("*.png", iconBytes)
	if err != nil {
		return errs.Wrap(err, "Could not write temp file")
	}

	notification := toast.Notification{
		AppID:   locale.T("org_name"),
		Title:   title,
		Icon:    iconFile,
		Message: message,
		Actions: []toast.Action{
			{"protocol", actionName, actionLink},
		},
		Duration: toast.Long,
	}
	if err := notification.Push(); err != nil {
		return errs.Wrap(err, "Could not send notification")
	}
	return nil
}
