package strutils

import (
	"bytes"
	"regexp"
	"strings"
	"text/template"
	"unicode"

	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"

	"github.com/ActiveState/cli/internal/errs"
)

func UUID() strfmt.UUID {
	return strfmt.UUID(uuid.New().String())
}

func ParseTemplate(contents string, params interface{}, funcMap template.FuncMap) (string, error) {
	tpl, err := template.New("template").Funcs(funcMap).Parse(contents)
	if err != nil {
		return "", errs.Wrap(err, "Could not parse template")
	}
	var out bytes.Buffer
	err = tpl.Execute(&out, params)
	if err != nil {
		return "", errs.Wrap(err, "Could not execute template")
	}
	return out.String(), nil
}

func Summarize(v string, maxLen int) string {
	if len(v) > maxLen {
		v = v[0:maxLen]
	}
	rx, err := regexp.Compile(`\s+`)
	if err == nil {
		v = rx.ReplaceAllString(v, " ")
	}
	v = strings.Replace(v, "\n", " ", -1)
	return v
}

func RemoveSpaces(v string) string {
	var b strings.Builder
	b.Grow(len(v))
	for _, ch := range v {
		if !unicode.IsSpace(ch) {
			b.WriteRune(ch)
		}
	}

	return b.String()
}
