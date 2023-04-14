package autostart

import (
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"howett.net/plist"
)

type LegacyPlist struct {
	ProgramArguments []string `plist:"ProgramArguments"`
}

func isLegacyPlist(path string) (bool, error) {
	if !fileutils.FileExists(path) {
		return false, nil
	}

	reader, err := os.Open(path)
	if err != nil {
		return false, errs.Wrap(err, "Could not open plist file")
	}

	decoder := plist.NewDecoder(reader)
	var p LegacyPlist
	err = decoder.Decode(&p)
	if err != nil {
		return false, errs.Wrap(err, "Could not decode plist file")
	}

	return len(p.ProgramArguments) > 1, nil
}
