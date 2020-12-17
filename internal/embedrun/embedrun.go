package embedrun

import (
	"fmt"
	"os/exec"

	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/gobuffalo/packr"
)

// Script runs the relevant script stored as an embedded asset.
func Script(scriptName, path string) error {
	box := packr.NewBox("../../assets/scripts/")
	scriptBlock := box.String(fmt.Sprintf("%s.bat", scriptName))
	sf, err := scriptfile.New(language.Batch, scriptName, scriptBlock)
	if err != nil {
		return err
	}

	cmd := exec.Command("cmd.exe", "/C", sf.Filename(), path)
	err = cmd.Start()
	if err != nil {
		return err
	}

	return nil
}
