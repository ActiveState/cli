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
	namespace := strings.TrimLeft(parsed.Path, "/")
	flag := fmt.Sprintf("--%s", parsed.Fragment)

	return runbits.InvokeSilent("activate", namespace, flag)
}
