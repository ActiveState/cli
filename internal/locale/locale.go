package locale

import (
	"os"
	"strings"

	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/nicksnyder/go-i18n/i18n"
	"github.com/thoas/go-funk"
)

// Supported languages
var Supported = []string{"en-US", "nl-NL"}

// T aliases to i18n.Tfunc()
var T func(translationID string, args ...interface{}) string

func init() {
	funk.ForEach(Supported, func(x string) {
		i18n.MustLoadTranslationFile("locale/" + strings.ToLower(x) + ".yaml")
	})

	Set("en-US")
}

// Set the active language to the given locale
func Set(localeName string) {
	if !funk.Contains(Supported, localeName) {
		print.Error("Locale does not exist: %s", localeName)
		os.Exit(1)
	}

	localT, _ := i18n.Tfunc(localeName)
	T = localT
}
