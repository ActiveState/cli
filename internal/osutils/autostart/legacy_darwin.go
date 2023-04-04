package autostart

import (
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"howett.net/plist"
)

type LegacyPlist struct {
	ProgramArguments []string `plist:"ProgramArguments"`
}

func isLegacyPlist(path string) (bool, error) {
	reader, err := os.Open(path)
	if err != nil {
		fmt.Println("error opening file: ", err)
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
