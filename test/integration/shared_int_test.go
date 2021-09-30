package integration

import (
	"fmt"
	"os"
	"runtime"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/strutils"
)

var (
	testUser          = "test-user"
	testProject       = "test-project"
	namespace         = fmt.Sprintf("%s/%s", testUser, testProject)
	url               = fmt.Sprintf("https://%s/%s", constants.PlatformURL, namespace)
	sampleYAMLPython2 = ""
	sampleYAMLPython3 = ""
	sampleYAMLEditor  = ""
)

func init() {
	if os.Getenv("VERBOSE") == "true" || os.Getenv("VERBOSE_TESTS") == "true" {
		logging.CurrentHandler().SetVerbose(true)
	}

	shell := "bash"
	if runtime.GOOS == "windows" {
		shell = "batch"
	}
	var err error
	box := packr.NewBox("../../assets/")
	sampleYAMLPython2, err = strutils.ParseTemplate(
		box.String("activestate.yaml.python.tpl"),
		map[string]interface{}{
			"Owner":    testUser,
			"Project":  testProject,
			"Shell":    shell,
			"Language": "python2",
			"LangExe":  language.MakeByName("python2").Executable().Filename(),
		})
	if err != nil {
		panic(err.Error())
	}
	sampleYAMLPython3, err = strutils.ParseTemplate(
		box.String("activestate.yaml.python.tpl"),
		map[string]interface{}{
			"Owner":    testUser,
			"Project":  testProject,
			"Shell":    shell,
			"Language": "python3",
			"LangExe":  language.MakeByName("python3").Executable().Filename(),
		})
	if err != nil {
		panic(err.Error())
	}
	sampleYAMLEditor, err = strutils.ParseTemplate(box.String("activestate.yaml.editor.tpl"), nil)
	if err != nil {
		panic(err.Error())
	}
}
