package strutils

import (
	"bytes"
	"regexp"
	"strings"
	"text/template"

	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"

	"github.com/ActiveState/cli/internal/errs"
)

func UUID() strfmt.UUID {
	return strfmt.UUID(uuid.New().String())
}

func ParseTemplate(contents string, params interface{}) (string, error) {
	tpl, err := template.New("template").Parse(contents)
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