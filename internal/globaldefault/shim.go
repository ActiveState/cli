package globaldefault

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	rt "runtime"
	"strings"

	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/strutils"
)

type shim struct {
	exePath string
	path    string
}

func newShim(exePath string) shim {
	target := filepath.Clean(filepath.Join(config.GlobalBinPath(), filepath.Base(exePath)))
	if rt.GOOS == "windows" {
		oldExt := filepath.Ext(target)
		target = target[0:len(target)-len(oldExt)] + ".bat"
	}
	return shim{exePath, target}
}

func (s shim) Create(languages []*language.Language) error {
	logging.Debug("Shimming %s at %s", s.exePath, s.path)

	// The link should not exist as we are always rolling back old shims before we run this code.
	if fileutils.TargetExists(s.path) {
		logging.Error("Shim already exists, forcefully overwriting: %s.", s.path)
		if err := os.Remove(s.path); err != nil {
			return locale.WrapError(err, "err_createshim_rm", "Could not remove old shim file: {{.V0}}.", s.path)
		}
	}

	exe, err := os.Executable()
	if err != nil {
		return errs.Wrap(err, "Could not get State Tool executable")
	}

	langs := []string{}
	for _, lang := range languages {
		langs = append(langs, lang.String())
	}

	tplParams := map[string]interface{}{
		"exe":       exe,
		"command":   filepath.Base(s.exePath),
		"languages": strings.Join(langs, ","),
		"denote":    shimDenoter,
	}
	box := packr.NewBox("../../assets/shim")
	boxFile := "shim.sh"
	if rt.GOOS == "windows" {
		boxFile = "shim.bat"
	}
	shimBytes := box.Bytes(boxFile)
	shimStr, err := strutils.ParseTemplate(string(shimBytes), tplParams)
	if err != nil {
		return errs.Wrap(err, "Could not parse %s template", boxFile)
	}

	err = ioutil.WriteFile(s.path, []byte(shimStr), 0755)
	if err != nil {
		return errs.Wrap(err, "failed to write shim command %s", s.path)
	}
	return nil
}

func (s shim) Path() string {
	return s.path
}

func (s shim) Languages() []*language.Language {
	contents, err := ioutil.ReadFile(s.path)
	if err != nil {
		logging.Debug("Could not read file contents of shim candidate %s: %v", s.path, err)
		return nil
	}

	targetRe := regexp.MustCompile(fmt.Sprintf("(?m)^(?:REM|#)%s: (.*)$", shimDenoter))
	target := targetRe.FindStringSubmatch(string(contents))

	if len(target) != 2 {
		logging.Error("Shim has no valid denotation: %s", s.path)
		return nil
	}

	languages := []*language.Language{}
	for _, shimLanguage := range strings.Split(target[1], ",") {
		lang := language.MakeByName(shimLanguage)
		languages = append(languages, &lang)
	}

	return languages
}

func (s shim) OneOfLanguage(languages []*language.Language) bool {
	for _, shimLang := range s.Languages() {
		for _, lang := range languages {
			if shimLang == lang {
				return true
			}
		}
	}
	return false
}
