package buildscript_runbit

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func generateDiff(script *buildscript.BuildScript, otherScript *buildscript.BuildScript) (string, error) {
	local := locale.Tl("diff_local", "local")
	remote := locale.Tl("diff_remote", "remote")

	var result bytes.Buffer

	sb1, err := script.Marshal()
	if err != nil {
		return "", errs.Wrap(err, "Could not marshal build script")
	}
	sb2, err := otherScript.Marshal()
	if err != nil {
		return "", errs.Wrap(err, "Could not marshal other build script")
	}

	diff := diffmatchpatch.New()
	scriptLines, newScriptLines, lines := diff.DiffLinesToChars(string(sb1), string(sb2))
	hunks := diff.DiffMain(scriptLines, newScriptLines, false)
	hunks = diff.DiffCharsToLines(hunks, lines)
	for i := 0; i < len(hunks); i++ {
		switch hunk := hunks[i]; hunk.Type {
		case diffmatchpatch.DiffEqual:
			result.WriteString(hunk.Text)
		case diffmatchpatch.DiffDelete:
			result.WriteString(fmt.Sprintf("<<<<<<< %s\n", local))
			result.WriteString(hunk.Text)
			result.WriteString("=======\n")
			if i+1 < len(hunks) && hunks[i+1].Type == diffmatchpatch.DiffInsert {
				result.WriteString(hunks[i+1].Text)
				i++ // do not process this hunk again
			}
			result.WriteString(fmt.Sprintf(">>>>>>> %s\n", remote))
		case diffmatchpatch.DiffInsert:
			result.WriteString(fmt.Sprintf("<<<<<<< %s\n", local))
			result.WriteString("=======\n")
			result.WriteString(hunk.Text)
			result.WriteString(fmt.Sprintf(">>>>>>> %s\n", remote))
		}
	}

	return result.String(), nil
}

func GenerateAndWriteDiff(proj *project.Project, script *buildscript.BuildScript, otherScript *buildscript.BuildScript) error {
	result, err := generateDiff(script, otherScript)
	if err != nil {
		return errs.Wrap(err, "Could not generate diff between local and remote build scripts")
	}
	return fileutils.WriteFile(filepath.Join(proj.Dir(), constants.BuildScriptFileName), []byte(result))
}
