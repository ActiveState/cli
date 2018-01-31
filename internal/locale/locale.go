package locale

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "github.com/ActiveState/ActiveState-CLI/internal/config"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/dvirsky/go-pylog/logging"
	"github.com/nicksnyder/go-i18n/i18n"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
)

// Supported languages
var Supported = []string{"en-US", "nl-NL"}

var translateFunction func(translationID string, args ...interface{}) string

var args = os.Args[1:]
var exit = os.Exit

func init() {
	logging.Debug("Init")

	viper.SetDefault("Locale", "en-US")

	path := getLocalePath()

	funk.ForEach(Supported, func(x string) {
		i18n.MustLoadTranslationFile(path + strings.ToLower(x) + ".yaml")
	})

	locale := getLocaleFlag()
	if locale == "" {
		locale = viper.GetString("Locale")
	}

	Set(locale)
}

// getLocalePath exists to facilitate running Go test scripts from their sub-directories, if no tests are being ran
// this just returns `locale/`
func getLocalePath() string {
	pathsep := string(os.PathSeparator)
	path := "locale" + pathsep

	if flag.Lookup("test.v") == nil {
		return path
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Panic("Could not call Caller(0)")
	}

	abs := filepath.Dir(file)

	// When tests are ran with coverage the location of this file is changed to a temp file, and we have to
	// adjust accordingly
	if strings.HasSuffix(abs, "_obj_test") {
		abs = ""
	}

	var err error
	abs, err = filepath.Abs(filepath.Join(abs, "..", ".."))

	if err != nil {
		log.Panic(err)
	}

	return abs + pathsep + path
}

// getLocaleFlag manually parses the input args looking for `--locale` or `-l` and retrieving its value
// this is done manually because Cobra depends on the localization being loaded
func getLocaleFlag() string {
	atValue := false

	for _, v := range args {
		if atValue {
			return v
		}
		if v == "--locale" || v == "-l" {
			atValue = true
		}
	}

	return ""
}

// Set the active language to the given locale
func Set(localeName string) {
	if !funk.Contains(Supported, localeName) {
		print.Error("Locale does not exist: %s", localeName)
		exit(1)
		return
	}

	translateFunction, _ = i18n.Tfunc(localeName)
	_ = translateFunction

	viper.Set("Locale", localeName)
}

// T aliases to i18n.Tfunc()
func T(translationID string, args ...interface{}) string {
	return translateFunction(translationID, args...)
}

// Tt aliases to T, but before returning the string it replaces `[[` and `]]` with `{{` and `}}`,
// allowing for the localized strings to use these template tags without triggering i18n
func Tt(translationID string, args ...interface{}) string {
	translation := translateFunction(translationID, args...)
	translation = strings.Replace(translation, "[[", "{{", -1)
	translation = strings.Replace(translation, "]]", "}}", -1)

	// For templates we want to manually specify the linebreaks as the way YAML gets parsed makes
	// this very painful otherwise
	translation = strings.Replace(translation, "\n", "", -1)
	translation = strings.Replace(translation, "{{BR}}", "\n", -1)

	translation = strings.Trim(translation, " ")
	return translation
}
