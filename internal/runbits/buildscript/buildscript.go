package buildscript

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func generateDiff(script *buildscript.Script, otherScript *buildscript.Script) (string, error) {
	local := locale.Tl("diff_local", "local")
	remote := locale.Tl("diff_remote", "remote")

	var result bytes.Buffer

	diff := diffmatchpatch.New()
	scriptLines, newScriptLines, lines := diff.DiffLinesToChars(script.String(), otherScript.String())
	hunks := diff.DiffMain(scriptLines, newScriptLines, false)
	hunks = diff.DiffCharsToLines(hunks, lines)
	hunks = diff.DiffCleanupSemantic(hunks)
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

func GenerateAndWriteDiff(proj *project.Project, script *buildscript.Script, otherScript *buildscript.Script) error {
	result, err := generateDiff(script, otherScript)
	if err != nil {
		return errs.Wrap(err, "Could not generate diff between local and remote build scripts")
	}
	return fileutils.WriteFile(filepath.Join(proj.Dir(), constants.BuildScriptFileName), []byte(result))
}
