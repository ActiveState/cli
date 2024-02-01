package projectmigration

import (
	"bytes"
	_ "embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/projectfile"
	"gopkg.in/yaml.v3"
)

//go:embed migrate.mac.bash
var migrateMacScript []byte

//go:embed migrate.nix.bash
var migrateNixScript []byte

//go:embed migrate.win.ps
var migrateWinScript []byte

type projecter interface {
	Source() *projectfile.Project
	Dir() string
	URL() string
	Path() string
	LegacyCommitID() string
	StripLegacyCommitID() error
}

var prompter prompt.Prompter
var out output.Outputer

// Register exists to avoid boilerplate in passing prompt and out to every caller of
// commitmediator.Get() for retrieving legacy commitId from activestate.yaml.
// This is an anti-pattern and is only used to make this legacy feature palatable.
func Register(prompter_ prompt.Prompter, out_ output.Outputer) {
	prompter = prompter_
	out = out_
}

func PromptAndMigrate(proj projecter) error {
	if prompter == nil || out == nil {
		return errs.New("projectmigration.Register() has not been called")
	}

	// We always set the local commit, the migration only touches on what happens with the commit in the activestate.yaml
	if err := localcommit.Set(proj.Dir(), proj.LegacyCommitID()); err != nil {
		return errs.Wrap(err, "Could not create local commit file")
	}
	for dir := proj.Dir(); filepath.Dir(dir) != dir; dir = filepath.Dir(dir) {
		if !fileutils.DirExists(filepath.Join(dir, ".git")) {
			continue
		}
		err := localcommit.AddToGitIgnore(dir)
		if err != nil {
			if !errors.Is(err, fs.ErrPermission) {
				multilog.Error("Unable to add local commit file to .gitignore: %v", err)
			}
			out.Notice(locale.T("notice_commit_id_gitignore"))
		}
		break
	}

	// Prevent also showing the warning when we already prompt
	warned = true

	// Skip full migration if env var is set
	if os.Getenv(constants.DisableProjectMigrationPrompt) == "true" {
		return nil
	}

	defaultChoice := false
	if migrate, err := prompter.Confirm("", locale.T("projectmigration_confirm"), &defaultChoice); err == nil && !migrate {
		if out.Config().Interactive {
			out.Notice(locale.T("projectmigration_declined"))
		}
		return CreateMigrateScript(proj)
	} else if err != nil {
		return errs.Wrap(err, "Could not confirm migration choice")
	}

	if err := proj.StripLegacyCommitID(); err != nil {
		return errs.Wrap(err, "Could not strip legacy commit ID")
	}

	out.Notice(locale.Tl("projectmigration_success", "Your project was successfully migrated"))

	return nil
}

var scriptsRx = regexp.MustCompile(`(?m)^scripts:\n`)

func CreateMigrateScript(proj projecter) error {
	scriptValue := migrateNixScript
	scriptLanguage := "bash"
	switch runtime.GOOS {
	case "darwin":
		scriptValue = migrateMacScript
	case "windows":
		scriptValue = migrateWinScript
		scriptLanguage = "powershell"
	}

	script := projectfile.Script{
		projectfile.NameVal{
			Name:  "migrate-to-buildscripts",
			Value: string(scriptValue),
		},
		projectfile.ScriptFields{
			Description: locale.T("projectmigration_script_description"),
			Standalone:  true,
			Language:    scriptLanguage,
		},
	}

	// We have to get a bit creative in writing the script, because calling `pjfile.Save()` will lead to reformatting
	// of user curated yaml.
	scriptB, err := yaml.Marshal(script)
	if err != nil {
		return errs.Wrap(err, "Could not marshal script")
	}

	// Indent our script block
	scriptB = bytes.Trim(scriptB, "\n")
	lines := bytes.Split(scriptB, []byte("\n"))
	for i, line := range lines {
		prefix := "    "
		if i == 0 {
			prefix = "  - "
		}
		lines[i] = append([]byte(prefix), line...)
	}
	scriptB = bytes.Join(lines, []byte("\n"))
	scriptB = append(scriptB, []byte("\n")...)

	// Splice it into the activestate.yaml
	asB, err := fileutils.ReadFile(proj.Source().Path())
	if err != nil {
		return errs.Wrap(err, "Could not read activestate.yaml")
	}

	scriptsPos := scriptsRx.FindIndex(asB)
	if scriptsPos != nil {
		asB = append(asB[:scriptsPos[1]], append(scriptB, asB[scriptsPos[1]:]...)...)
	} else {
		asB = append(asB, append([]byte("\nscripts:\n"), scriptB...)...)
	}

	if err := fileutils.WriteFile(proj.Source().Path(), asB); err != nil {
		return errs.Wrap(err, "Could not write activestate.yaml")
	}

	return nil
}

// Only show once per state tool invocation
var warned = false

func Warn(proj projecter) error {
	if warned {
		return nil
	}

	if prompter == nil || out == nil {
		return errs.New("projectmigration.Register() has not been called")
	}

	warned = true

	out.Notice(locale.Tr("projectmigration_warning", proj.Source().Path()))

	return nil
}
