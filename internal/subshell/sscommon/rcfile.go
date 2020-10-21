package sscommon

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gobuffalo/packr"
	"github.com/mash/go-tempfile-suffix"
	"github.com/spf13/viper"

	"github.com/ActiveState/cli/internal/colorize"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/output/txtstyle"
	"github.com/ActiveState/cli/pkg/project"
)

var (
	Deploy EnvData = EnvData{
		constants.RCAppendDeployStartLine,
		constants.RCAppendDeployStopLine,
		"user_env",
	}
	Default EnvData = EnvData{
		constants.RCAppendDefaultStartLine,
		constants.RCAppendDefaultStopLine,
		"user_default_env",
	}
)

type EnvData struct {
	Start string
	Stop  string
	Key   string
}

func WriteRcFile(rcTemplateName string, path string, data EnvData, env map[string]string) *failures.Failure {
	if fail := fileutils.Touch(path); fail != nil {
		return fail
	}

	rcData := map[string]interface{}{
		"Start": data.Start,
		"Stop":  data.Stop,
		"Env":   env,
	}

	if fail := cleanRcFile(path, data); fail != nil {
		return fail
	}

	box := packr.NewBox("../../../assets/shells")
	tpl := box.String(rcTemplateName)

	t, err := template.New("rcfile_append").Parse(tpl)
	if err != nil {
		return failures.FailTemplating.Wrap(err)
	}

	var out bytes.Buffer
	err = t.Execute(&out, rcData)
	if err != nil {
		return failures.FailTemplating.Wrap(err)
	}

	logging.Debug("Writing to %s:\n%s", path, out.String())

	return fileutils.AppendToFile(path, []byte(fileutils.LineEnd+out.String()))
}

func cleanRcFile(path string, data EnvData) *failures.Failure {
	readFile, err := os.Open(path)

	if err != nil {
		return failures.FailIO.Wrap(err)
	}

	scanner := bufio.NewScanner(readFile)
	scanner.Split(bufio.ScanLines)

	var strip bool
	var fileContents []string
	for scanner.Scan() {
		text := scanner.Text()

		// Detect start line
		if strings.Contains(text, data.Start) {
			logging.Debug("Cleaning previous RC lines from %s", path)
			strip = true
		}

		// Strip line
		if strip {
			continue
		}

		// Rebuild file contents
		fileContents = append(fileContents, scanner.Text())

		// Detect stop line
		if strings.Contains(text, data.Stop) {
			strip = false
		}
	}
	readFile.Close()

	return fileutils.WriteFile(path, []byte(strings.Join(fileContents, fileutils.LineEnd)))
}

// SetupShellRcFile create a rc file to activate a runtime (without a project being present)
func SetupShellRcFile(rcFileName, templateName string, env map[string]string, namespace project.Namespaced) error {
	box := packr.NewBox("../../../assets/shells")
	tpl := box.String(templateName)

	rcData := map[string]interface{}{
		"Env":     env,
		"Project": namespace.String(),
	}
	t, err := template.New("rcfile").Parse(tpl)
	if err != nil {
		return errs.Wrap(err, "Failed to parse template file.")
	}

	var out bytes.Buffer
	err = t.Execute(&out, rcData)
	if err != nil {
		return errs.Wrap(err, "failed to execute template.")
	}

	f, err := os.Create(rcFileName)
	if err != nil {
		return locale.WrapError(err, "sscommon_rc_file_creation_err", "Failed to create file {{.V0}}", rcFileName)
	}
	defer f.Close()

	f.WriteString(out.String())

	err = os.Chmod(rcFileName, 0755)
	if err != nil {
		return errs.Wrap(err, "Failed to set executable flag.")
	}
	return nil
}

// SetupProjectRcFile creates a temporary RC file that our shell is initiated from, this allows us to template the logic
// used for initialising the subshell
func SetupProjectRcFile(templateName, ext string, env map[string]string, out output.Outputer) (*os.File, *failures.Failure) {
	box := packr.NewBox("../../../assets/shells")
	tpl := box.String(templateName)
	prj := project.Get()

	userScripts := ""
	activatedKey := fmt.Sprintf("activated_%s", prj.Namespace().String())
	for _, event := range prj.Events() {
		v, err := event.Value()
		if err != nil {
			return nil, failures.FailMisc.Wrap(err)
		}
		if strings.ToLower(event.Name()) == "first-activate" && !viper.GetBool(activatedKey) {
			userScripts = v + "\n" + userScripts
			viper.Set(activatedKey, true)
		}
		if event.Name() == "activate" {
			userScripts = userScripts + "\n" + v
		}
	}

	inuse := []string{}
	scripts := map[string]string{}
	var explicitName string

	// Prepare script map to be parsed by template
	for _, cmd := range prj.Scripts() {
		explicitName = fmt.Sprintf("%s_%s", prj.NormalizedName(), cmd.Name())

		_, err := exec.LookPath(cmd.Name())
		if err == nil {
			// Do not overwrite commands that are already in use and
			// keep track of those commands to warn to the user
			inuse = append(inuse, cmd.Name())
			continue
		}

		scripts[cmd.Name()] = cmd.Name()
		scripts[explicitName] = cmd.Name()
	}

	if len(inuse) > 0 {
		out.Notice(output.Heading(locale.Tl("warn_scriptinuse_title", "Warning: Script Names Already In Use")))
		out.Notice(locale.Tr("warn_script_name_in_use", strings.Join(inuse, "\n  [DISABLED]-[/RESET] "), inuse[0], explicitName))
	}

	wd, err := osutils.Getwd()
	if err != nil {
		return nil, failures.FailMisc.Wrap(err, locale.Tr("err_subshell_wd", "Could not get working directory."))
	}

	isConsole := ext == ".bat" // yeah this is a dirty cheat, should find something more deterministic

	activatedMessage := txtstyle.NewTitle(locale.Tl("youre_activated", "You're Activated!"))
	activatedMessage.ColorCode = "SUCCESS"

	var activateEvtMessage string
	if userScripts != "" {
		activateEvtMessage = output.Heading(locale.Tl("activate_event_message", "Running Activation Events")).String()
	}

	rcData := map[string]interface{}{
		"Owner":                prj.Owner(),
		"Name":                 prj.Name(),
		"Env":                  env,
		"WD":                   wd,
		"UserScripts":          userScripts,
		"Scripts":              scripts,
		"ExecName":             constants.CommandName,
		"ActivatedMessage":     "\n" + colorize.ColorizedOrStrip(activatedMessage.String(), isConsole),
		"ActivateEventMessage": colorize.ColorizedOrStrip(activateEvtMessage, isConsole),
	}

	currExecAbsPath, err := osutils.Executable()
	if err != nil {
		return nil, failures.FailOS.Wrap(err)
	}
	currExecAbsDir := filepath.Dir(currExecAbsPath)

	listSep := string(os.PathListSeparator)
	pathList, ok := env["PATH"]
	inPathList, err := fileutils.PathInList(listSep, pathList, currExecAbsDir)
	if !ok || !inPathList {
		rcData["ExecAlias"] = currExecAbsPath // alias {ExecName}={ExecAlias}
	}

	t, err := template.New("rcfile").Parse(tpl)
	if err != nil {
		return nil, failures.FailTemplating.Wrap(err)
	}

	var o bytes.Buffer
	err = t.Execute(&o, rcData)
	if err != nil {
		return nil, failures.FailTemplating.Wrap(err)
	}

	tmpFile, err := tempfile.TempFileWithSuffix(os.TempDir(), "state-subshell-rc", ext)
	if err != nil {
		return nil, failures.FailOS.Wrap(err)
	}
	defer tmpFile.Close()

	tmpFile.WriteString(o.String())

	logging.Debug("Using project RC: %s", o.String())

	return tmpFile, nil
}
