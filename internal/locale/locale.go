package locale

import (
	"os"
	"strings"

	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/dvirsky/go-pylog/logging"
	"github.com/nicksnyder/go-i18n/i18n"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
)

// Supported languages
var Supported = []string{"en-US", "nl-NL"}

var translateFunction func(translationID string, args ...interface{}) string

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

func Init() {
	logging.Debug("Init")

	viper.SetDefault("Locale", "en-US")

	funk.ForEach(Supported, func(x string) {
		i18n.MustLoadTranslationFile("locale/" + strings.ToLower(x) + ".yaml")
	})

	Set(viper.GetString("Locale"))
}

// Set the active language to the given locale
func Set(localeName string) {
	if !funk.Contains(Supported, localeName) {
		print.Error("Locale does not exist: %s", localeName)
		os.Exit(1)
	}

	translateFunction, _ = i18n.Tfunc(localeName)
	_ = translateFunction

	viper.Set("Locale", localeName)
}
