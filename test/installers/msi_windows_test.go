package installers_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/termtest"
)

var (
	rootDir = `Z:\bin`
	asToken = "ActiveState"

	perlLang = language{
		name:     "perl",
		checkCmd: "perl -v",
	}

	perlMsiFileNames = []string{
		"ActivePerl-5.26.msi",
		//"ActivePerl-5.28.msi",
	}

	installAction   msiExecAction = "install"
	uninstallAction msiExecAction = "uninstall"
)

type language struct {
	name     string
	checkCmd string
}

type msiFile struct {
	path    string
	version string
	lang    language
}

func newMsiFile(filename string, lang language) *msiFile {
	return &msiFile{
		path:    filepath.Join(rootDir, filename),
		version: versionFromMsiFileName(filename),
		lang:    lang,
	}
}

type msiExecAction string

func (a msiExecAction) cmdText(msiPath string) string {
	msiAct := "/package"
	if a == uninstallAction {
		msiAct = "/uninstall"
	}

	form := `Start-Process msiexec.exe -Wait -ArgumentList "%s %s /quiet /qn /norestart /log %s" -PassThru`
	return fmt.Sprintf(form, msiPath, msiAct, a.logFileName(msiPath))
}

func (a msiExecAction) logFileName(msiPath string) string {
	msiName := filepath.Base(strings.TrimSuffix(msiPath, filepath.Ext(msiPath)))
	return fmt.Sprintf(`%s\%s_%s.log`, rootDir, msiName, string(a))
}

func versionFromMsiFileName(name string) string {
	name = strings.TrimSuffix(name, filepath.Ext(name))
	i := strings.LastIndexByte(name, '-')
	return name[i+1:]
}

type psSession struct {
	*termtest.ConsoleProcess
}

func newPSSession(root string) (*psSession, error) {
	opts := termtest.Options{
		CmdName:       "powershell",
		WorkDirectory: root,
		RetainWorkDir: true,
	}
	cp, err := termtest.New(opts)
	if err != nil {
		return nil, err
	}

	s := psSession{
		ConsoleProcess: cp,
	}

	return &s, nil
}

func (s *psSession) Expect(fail func(...interface{}), value string) {
	if out, err := s.ConsoleProcess.Expect(value); err != nil {
		fail(err, out)
	}
}

func (s *psSession) ExpectNone(fail func(...interface{}), values ...string) {
	trimmed := s.ConsoleProcess.TrimmedSnapshot()
	for _, val := range values {
		if strings.Contains(trimmed, val) {
			fail(fmt.Sprintf("incorrectly contains: %s", val))
		}
	}
}

func TestActivePerl(t *testing.T) {
	for _, msiFileName := range perlMsiFileNames {
		t.Run(msiFileName, func(t *testing.T) {
			m := newMsiFile(msiFileName, perlLang)
			cp, err := newPSSession(rootDir)
			if err != nil {
				t.Fatal(err)
			}

			cp.Send(installAction.cmdText(m.path))
			cp.Send("echo $?")
			cp.Expect(t.Fatal, "True")

			cp.Send(m.lang.checkCmd)
			cp.Expect(t.Error, m.version)
			cp.Expect(t.Error, asToken)

			cp.Send(uninstallAction.cmdText(m.path))
			cp.Send("echo $?")
			cp.Expect(t.Fatal, "True")

			cp.Send(m.lang.checkCmd)
			cp.ExpectNone(t.Fatal, asToken)
		})
	}
}
