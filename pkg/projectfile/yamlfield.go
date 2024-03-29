package projectfile

import (
	"bytes"
	"fmt"
	"os"
	"regexp"

	"github.com/ActiveState/cli/internal/errs"
	"gopkg.in/yaml.v2"
)

type yamlField struct {
	field string
	value interface{}
}

func NewYamlField(field string, value interface{}) *yamlField {
	return &yamlField{field: field, value: value}
}

func (y *yamlField) Save(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return errs.Wrap(err, "ioutil.ReadFile %s failed", path)
	}

	lineSep := []byte("\n")
	if bytes.Contains(data, []byte("\r\n")) {
		lineSep = []byte("\r\n")
	}

	var re = regexp.MustCompile(fmt.Sprintf(`(?m:^%s:\s+?(.*?)$)`, regexp.QuoteMeta(y.field)))
	addLine, err := yaml.Marshal(map[string]interface{}{y.field: y.value})
	addLine = bytes.TrimRight(addLine, string(lineSep))

	out := re.ReplaceAll(data, addLine)
	if !bytes.Contains(out, addLine) {
		// Nothing to replace; append to the end of the file instead
		addLine = append(lineSep, addLine...) // Prepend line ending
		addLine = append(addLine, lineSep...) // Append line ending
		out = append(bytes.TrimRight(out, string(lineSep)), addLine...)
	}

	if err := os.WriteFile(path, out, 0664); err != nil {
		return errs.Wrap(err, "ioutil.WriteFile %s failed", path)
	}

	return nil
}
