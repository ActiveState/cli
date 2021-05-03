package strutils

import (
	"bytes"
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
