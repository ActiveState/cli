package protocol

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/project"
)

type primeable interface {
	primer.Outputer
}

type Protocol struct {
	out output.Outputer
}

type Params struct {
	URL string
}

func New(prime primeable) *Protocol {
	return &Protocol{prime.Output()}
}

func (p *Protocol) Run(params Params) error {
	logging.Debug("Execute Protocol")

	parsed, err := url.Parse(params.URL)
	if err != nil {
		return locale.WrapError(err, "err_protocol_parse", "Invailid URL provided: {{.V0}}", params.URL)
	}
	trimmedPath := strings.TrimLeft(parsed.Path, "/")
	namespace, fail := project.ParseNamespace(trimmedPath)
	if fail != nil {
		return locale.WrapError(fail, "err_protocol_namespace", "{{.V0}} is not a valid namespace", trimmedPath)
	}

	if parsed.Fragment == "" || parsed.Fragment != "replace" {
		return locale.NewError("err_protocol_flag", "Invalid URL fragment, the only supported URL fragment is 'replace'")
	}

	return runbits.InvokeSilent("activate", namespace.String(), fmt.Sprintf("--%s", parsed.Fragment))
}
