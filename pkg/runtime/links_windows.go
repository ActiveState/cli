package runtime

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/smartlink"
)

const linkTarget = "__target__"
const link = "__link__"

func supportsHardLinks(path string) (supported bool) {
	defer func() {
		if !supported {
			logging.Debug("Enforcing deployment via copy, as hardlinks are not supported")
		}
	}()

	target := filepath.Join(path, linkTarget)
	err := fileutils.Touch(target)
	if err != nil {
		multilog.Error("Error touching target: %v", err)
		return false
	}
	defer func() {
		err := os.Remove(target)
		if err != nil {
			multilog.Error("Error removing target: %v", err)
		}
	}()

	lnk := filepath.Join(path, link)
	if fileutils.TargetExists(lnk) {
		err := os.Remove(lnk)
		if err != nil {
			multilog.Error("Error removing previous link: %v", err)
			return false
		}
	}

	logging.Debug("Attempting to link '%s' to '%s'", lnk, target)
	err = smartlink.Link(target, lnk)
	if err != nil {
		logging.Debug("Test link creation failed: %v", err)
		return false
	}
	err = os.Remove(lnk)
	if err != nil {
		multilog.Error("Error removing link: %v", err)
	}

	return true
}
