package locale

// This package may NOT depend on failures (directly or indirectly)

import (
	"bytes"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/nicksnyder/go-i18n/i18n"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

// Supported languages
var Supported = []string{"en-US", "nl-NL"}

var translateFunction func(translationID string, args ...interface{}) string

var args = os.Args[1:]
var exit = os.Exit

//go:embed locales
var localeFiles embed.FS

func init() {
	defer profile.Measure("locale:init", time.Now())
	logging.Debug("Init")

	locale := getLocaleFlag()
	if locale == "" {
		cfg, err := config.New()
		if err == nil {
			locale = cfg.GetString("Locale")
			if locale == "" {
				locale = "en-US"
				setErr := cfg.Set("Locale", locale)
				if setErr != nil {
					logging.Error("Could not set locale entry in config, error: %v", setErr)
				}
			}
		} else {
			logging.Error("Could not load  config to check locale, error: %v", err)
			locale = "en-US"
		}
	}

	localePath := getLocalePath()

	funk.ForEach(Supported, func(x string) {
		localeFile := strings.ToLower(x)+".yaml")
		bytes, err := localeFiles.ReadFile("locales/"+localeFile) // need '/' for go:embed, even on Windows
		if err != nil {
			panic(fmt.Sprintf("Could not read asset %s: %v", localeFile, err))
		}
		path := filepath.Join(localePath, localeFile)
		err = i18n.ParseTranslationFileBytes(path, bytes)
		if err != nil {
			panic(fmt.Sprintf("Could not load %s: %v", path, err))
		}
	})

	if err := Set(locale); err != nil {
		logging.Error("Could not set locale: %v", err)
	}
}

// getLocalePath exists to facilitate running Go test scripts from their sub-directories, if no tests are being run
// this just returns `internal/locale/`
func getLocalePath() string {
	rootpath, err := environment.GetRootPath()
	if err != nil {
		log.Panic(err)
		return ""
	}
	return filepath.Join(rootpath, "internal", "locale")
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
func Set(localeName string) (rerr error) {
	if !funk.Contains(Supported, localeName) {
		return errs.New("Locale does not exist: %s", localeName)
	}

	translateFunction, _ = i18n.Tfunc(localeName)
	_ = translateFunction

	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not get configuration to store updated locale")
	}
	defer rtutils.Closer(cfg.Close, &rerr)

	err = cfg.Set("Locale", localeName)
	if err != nil {
		return errs.Wrap(err, "Could not set locale in config")
	}

	return nil
}

// T aliases to i18n.Tfunc()
func T(translationID string, args ...interface{}) string {
	if translateFunction == nil {
		return translationID
	}
	return translateFunction(translationID, args...)
}

// Tr is like T but it accepts string params that will be used as numbered params, eg. V0, V1, V2 etc
func Tr(translationID string, values ...string) string {
	return T(translationID, parseInput(values...))
}

// Tl is like Tr but it accepts a fallback locale for if the translationID is not found
func Tl(translationID, locale string, values ...string) string {
	translation := Tr(translationID, values...)

	if translation == translationID {
		translation = locale
		input := parseInput(values...)

		// prepare template
		tmpl, err := template.New("locale error").Parse(translation)
		if err != nil {
			logging.Error("Invalid translation template: %w", err)
			return translation
		}

		// parse template
		var out bytes.Buffer
		err = tmpl.Execute(&out, input)
		if err != nil {
			logging.Error("Could not execute translation template: %w", err)
			return translation
		}
		translation = out.String()
	}

	return translation
}

func parseInput(values ...string) map[string]interface{} {
	var input = map[string]interface{}{}
	for k, v := range values {
		input["V"+strconv.Itoa(k)] = v
	}
	return input
}

// Tt aliases to T, but before returning the string it replaces `[[` and `]]` with `{{` and `}}`,
// allowing for the localized strings to use these template tags without triggering i18n
func Tt(translationID string, args ...interface{}) string {
	if translateFunction == nil {
		return translationID
	}
	translation := translateFunction(translationID, args...)
	translation = strings.Replace(translation, "[[", "{{", -1)
	translation = strings.Replace(translation, "]]", "}}", -1)

	// For templates we want to manually specify the linebreaks as the way YAML gets parsed makes
	// this very painful otherwise

	// Replace newlines in yaml strings with space to avoid concatenated words
	replaceRegex := regexp.MustCompile(`\s*\n`)
	translation = replaceRegex.ReplaceAllString(translation, " ")
	translation = strings.Replace(translation, "{{BR}}", "\n", -1)
	// Avoid indentation after newlines
	translation = strings.Replace(translation, "\n ", "\n", -1)

	translation = strings.Trim(translation, " ")
	return translation
}
