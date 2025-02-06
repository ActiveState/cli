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

	"github.com/mash/go-tempfile-suffix"

	"github.com/ActiveState/cli/internal/installation/storage"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/colorize"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/project"
)

var (
	DeployID RcIdentification = RcIdentification{
		constants.RCAppendDeployStartLine,
		constants.RCAppendDeployStopLine,
		"user_env",
	}
	DefaultID RcIdentification = RcIdentification{
		constants.RCAppendDefaultStartLine,
		constants.RCAppendDefaultStopLine,
		"user_default_env",
	}
	InstallID RcIdentification = RcIdentification{
		constants.RCAppendInstallStartLine,
		constants.RCAppendInstallStopLine,
		"user_install_env",
	}
	OfflineInstallID RcIdentification = RcIdentification{
		constants.RCAppendOfflineInstallStartLine,
		constants.RCAppendOfflineInstallStopLine,
		"user_offlineinstall_env",
	}
	AutostartID RcIdentification = RcIdentification{
		constants.RCAppendAutostartStartLine,
		constants.RCAppendAutostartStopLine,
		"user_autostart_env",
	}
)

// Configurable defines an interface to store and get configuration data
type Configurable interface {
	Set(string, interface{}) error
	GetBool(string) bool
	GetString(string) string
	GetStringMap(string) map[string]interface{}
}

type RcIdentification struct {
	Start string
	Stop  string
	Key   string
}

func WriteRcFile(rcTemplateName string, path string, data RcIdentification, env map[string]string) error {
	if err := fileutils.Touch(path); err != nil {
		return err
	}

	rcData := map[string]interface{}{
		"Start":                 data.Start,
		"Stop":                  data.Stop,
		"Env":                   env,
		"ActivatedEnv":          constants.ActivatedStateEnvVarName,
		"ConfigFile":            constants.ConfigFileName,
		"ActivatedNamespaceEnv": constants.ActivatedStateNamespaceEnvVarName,
		"Default":               data == DefaultID,
	}

	if err := CleanRcFile(path, data); err != nil {
		return err
	}

	tpl, err := assets.ReadFileBytes(fmt.Sprintf("shells/%s", rcTemplateName))
	if err != nil {
		return errs.Wrap(err, "Failed to read asset")
	}
	t, err := template.New("rcfile_append").Parse(string(tpl))
	if err != nil {
		return errs.Wrap(err, "Templating failure")
	}

	var out bytes.Buffer
	err = t.Execute(&out, rcData)
	if err != nil {
		return errs.Wrap(err, "Templating failure")
	}

	return fileutils.AppendToFile(path, []byte(fileutils.LineEnd+out.String()))
}

func WriteRcData(data string, path string, identification RcIdentification) error {
	if err := fileutils.Touch(path); err != nil {
		return err
	}

	if err := CleanRcFile(path, identification); err != nil {
		return err
	}

	data = identification.Start + fileutils.LineEnd + data + fileutils.LineEnd + identification.Stop
	return fileutils.AppendToFile(path, []byte(fileutils.LineEnd+data))
}

// RemoveLegacyInstallPath removes the PATH modification statement added to the shell-rc file by the legacy install script
func RemoveLegacyInstallPath(path string) error {
	if err := fileutils.Touch(path); err != nil {
		return err
	}
	readFile, err := os.Open(path)
	if err != nil {
		return errs.Wrap(err, "IO failure")
	}

	scanner := bufio.NewScanner(readFile)
	scanner.Split(bufio.ScanLines)

	var fileContents []string
	for scanner.Scan() {
		text := scanner.Text()

		// remove lines with marker added by legacy install script
		if strings.Contains(text, "# ActiveState State Tool") {
			continue
		}

		// Rebuild file contents
		fileContents = append(fileContents, scanner.Text())
	}
	if err := readFile.Close(); err != nil {
		return errs.Wrap(err, "failed to close %s", path)
	}

	return fileutils.WriteFile(path, []byte(strings.Join(fileContents, fileutils.LineEnd)))
}

func CleanRcFile(path string, data RcIdentification) error {
	if err := fileutils.Touch(path); err != nil {
		return err
	}
	readFile, err := os.Open(path)
	if err != nil {
		return errs.Wrap(err, "IO failure")
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

		// Detect stop line
		if strings.Contains(text, data.Stop) {
			strip = false
			continue
		}

		// Strip line
		if strip {
			continue
		}

		// Rebuild file contents
		fileContents = append(fileContents, scanner.Text())
	}
	readFile.Close()

	return fileutils.WriteFile(path, []byte(strings.Join(fileContents, fileutils.LineEnd)))
}

// SetupShellRcFile create a rc file to activate a runtime (without a project being present)
func SetupShellRcFile(rcFileName, templateName string, env map[string]string, namespace *project.Namespaced, cfg Configurable) error {
	tpl, err := assets.ReadFileBytes(fmt.Sprintf("shells/%s", templateName))
	if err != nil {
		return errs.Wrap(err, "Failed to read asset")
	}
	t, err := template.New("rcfile").Parse(string(tpl))
	if err != nil {
		return errs.Wrap(err, "Failed to parse template file.")
	}

	projectValue := ""
	if namespace != nil {
		projectValue = namespace.String()
	}

	var out bytes.Buffer
	rcData := map[string]interface{}{
		"Env":     env,
		"Project": projectValue,
	}
	err = t.Execute(&out, rcData)
	if err != nil {
		return errs.Wrap(err, "failed to execute template.")
	}

	f, err := os.Create(rcFileName)
	if err != nil {
		return locale.WrapError(err, "sscommon_rc_file_creation_err", "Failed to create file {{.V0}}", rcFileName)
	}
	defer f.Close()

	_, err = f.WriteString(out.String())
	if err != nil {
		return errs.Wrap(err, "Failed to write to output buffer.")
	}

	err = os.Chmod(rcFileName, 0755)
	if err != nil {
		return errs.Wrap(err, "Failed to set executable flag.")
	}
	return nil
}

// SetupProjectRcFile creates a temporary RC file that our shell is initiated from, this allows us to template the logic
// used for initialising the subshell
func SetupProjectRcFile(prj *project.Project, templateName, ext string, env map[string]string, out output.Outputer, cfg Configurable, bashifyPaths bool) (*os.File, error) {
	tpl, err := assets.ReadFileBytes(fmt.Sprintf("shells/%s", templateName))
	if err != nil {
		return nil, errs.Wrap(err, "Failed to read asset")
	}

	userScripts := ""

	// Yes this is awkward, issue here - https://www.pivotaltracker.com/story/show/175619373
	activatedKey := fmt.Sprintf("activated_%s", prj.Namespace().String())
	for _, eventType := range project.ActivateEvents() {
		event := prj.EventByName(eventType.String(), bashifyPaths)
		if event == nil {
			continue
		}

		v, err := event.Value()
		if err != nil {
			return nil, errs.Wrap(err, "Could not get event value")
		}

		if strings.ToLower(event.Name()) == project.FirstActivate.String() && !cfg.GetBool(activatedKey) {
			userScripts = v + "\n" + userScripts
		}

		if strings.ToLower(event.Name()) == project.Activate.String() {
			userScripts = userScripts + "\n" + v
		}
	}
	err = cfg.Set(activatedKey, true)
	if err != nil {
		return nil, errs.Wrap(err, "Could not set activatedKey in config")
	}

	inuse := []string{}
	scripts := map[string]string{}
	var explicitName string
	globalBinDir := filepath.Clean(storage.GlobalBinDir())

	// Prepare script map to be parsed by template
	projectScripts, err := prj.Scripts()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get project scripts")
	}
	for _, cmd := range projectScripts {
		explicitName = fmt.Sprintf("%s_%s", prj.NormalizedName(), cmd.Name())

		path, err := exec.LookPath(cmd.Name())
		dir := filepath.Clean(filepath.Dir(path))
		if dir == globalBinDir {
			continue
		}
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
		out.Notice(locale.Tr("warn_script_name_in_use", strings.Join(inuse, "[/RESET],[NOTICE] "), inuse[0], explicitName))
	}

	wd, err := osutils.Getwd()
	if err != nil {
		return nil, locale.WrapError(err, "err_subshell_wd", "", "Could not get working directory.")
	}

	isConsole := ext == ".bat" // yeah this is a dirty cheat, should find something more deterministic

	actualEnv := map[string]string{}
	for k, v := range env {
		if strings.Contains(v, "\n") {
			logging.Warning("Env key %s has a multi-line value, which is not supported", k)
			continue
		}
		actualEnv[k] = v
	}

	rcData := map[string]interface{}{
		"Owner":       prj.Owner(),
		"Name":        prj.Name(),
		"Project":     prj.NamespaceString(),
		"Env":         actualEnv,
		"WD":          wd,
		"UserScripts": userScripts,
		"Scripts":     scripts,
		"ExecName":    constants.CommandName,
		"ActivatedMessage": colorize.ColorizedOrStrip(locale.Tl("project_activated",
			"[SUCCESS]✔ Project \"{{.V0}}\" Has Been Activated[/RESET]", prj.Namespace().String()), isConsole),
	}

	currExec := osutils.Executable()
	currExecAbsDir := filepath.Dir(currExec)
	if bashifyPaths {
		currExec, err = osutils.BashifyPath(currExec)
		if err != nil {
			return nil, errs.Wrap(err, "Could not bashify executable: %s", currExec)
		}
	}

	listSep := string(os.PathListSeparator)
	pathList, ok := env["PATH"]
	inPathList, err := fileutils.PathInList(listSep, pathList, currExecAbsDir)
	if err != nil {
		return nil, errs.Wrap(err, "Could not check if %s is in PATH", currExecAbsDir)
	}
	if !ok || !inPathList {
		safeExec := currExec
		if strings.ContainsAny(currExec, " ") {
			safeExec = fmt.Sprintf(`"%s"`, currExec) // quote for alias
		}
		rcData["ExecAlias"] = safeExec // alias {ExecName}={ExecAlias}
	}

	t := template.New("rcfile")
	t.Funcs(map[string]interface{}{
		"splitLines": func(v string) []string { return strings.Split(v, "\n") },
		"escapePwsh": func(v string) string {
			// Conver unicode characters
			result := ""
			for _, char := range v {
				if char < 128 {
					result += string(char)
				} else {
					result += fmt.Sprintf("$([char]0x%04x)", char)
				}
			}

			// Escape special characters
			result = strings.ReplaceAll(result, "`", "``")
			result = strings.ReplaceAll(result, "\"", "`\"")
			return result
		},
	})

	t, err = t.Parse(string(tpl))
	if err != nil {
		return nil, errs.Wrap(err, "Templating failure")
	}

	var o bytes.Buffer
	err = t.Execute(&o, rcData)
	if err != nil {
		return nil, errs.Wrap(err, "Templating failure")
	}

	tmpFile, err := tempfile.TempFileWithSuffix(os.TempDir(), "state-subshell-rc", ext)
	if err != nil {
		return nil, errs.Wrap(err, "OS failure")
	}
	defer tmpFile.Close()

	_, err = tmpFile.WriteString(o.String())
	if err != nil {
		return nil, errs.Wrap(err, "Failed to write to output buffer.")
	}

	return tmpFile, nil
}

func ProjectRCIdentifier(base RcIdentification, namespace *project.Namespaced) RcIdentification {
	id := base
	id.Start = fmt.Sprintf("%s-%s", id.Start, namespace.String())
	id.Stop = fmt.Sprintf("%s-%s", id.Stop, namespace.String())
	id.Key = fmt.Sprintf("%s_%s", id.Key, namespace.String())
	return id
}
