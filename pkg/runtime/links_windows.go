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
	logging.Debug("Determining if hard links are supported for drive associated with '%s'", path)
	defer func() {
		log := "Yes they are"
		if !supported {
			log = "No they are not"
		}
		logging.Debug(log)
	}()

	target := filepath.Join(path, linkTarget)
	if !fileutils.TargetExists(target) {
		err := fileutils.Touch(target)
		if err != nil {
			multilog.Error("Error touching target: %v", err)
			return false
		}
	}

	lnk := filepath.Join(path, link)
	if fileutils.TargetExists(lnk) {
		err := os.Remove(lnk)
		if err != nil {
			multilog.Error("Error removing previous link: %v", err)
			return false
		}
	}

	logging.Debug("Attempting to link '%s' to '%s'", lnk, target)
	err := smartlink.Link(target, lnk)
	if err != nil {
		logging.Debug("Test link creation failed: %v", err)
	}
	return err == nil
}
